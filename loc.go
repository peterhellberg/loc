package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/antage/eventsource"
	"github.com/oschwald/maxminddb-golang"
	"github.com/rwynn/gtm"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
)

var visitsNamespace string

// NewVisits filters on inserts to the visits collection
func NewVisits(op *gtm.Op) bool {
	return op.Namespace == visitsNamespace && op.IsInsert()
}

type visitEvent struct {
	Location struct {
		Latitude  float64 `maxminddb:"latitude"`
		Longitude float64 `maxminddb:"longitude"`
	} `maxminddb:"location"`

	City struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"city"`
	Country struct {
		IsoCode string `maxminddb:"iso_code"`
	} `maxminddb:"country"`
}

func (v *visitEvent) CityName(lang string) string {
	return v.City.Names[lang]
}

func (v *visitEvent) HasEnglishCityName() bool {
	return v.CityName("en") != ""
}

func (v *visitEvent) InScandinavia() bool {
	return v.Country.IsoCode == "SE" ||
		v.Country.IsoCode == "NO" ||
		v.Country.IsoCode == "DK" ||
		v.Country.IsoCode == "FI" ||
		v.Country.IsoCode == "AX"
}

func (v *visitEvent) JSON(id string) string {
	return fmt.Sprintf(`{"city":"%s","iso_code":"%s","lat":%v,"long":%v,"id":"%s"}`,
		v.CityName("en"), v.Country.IsoCode,
		v.Location.Latitude, v.Location.Longitude, id)
}

func main() {
	// Read the GeoLite2 City database into memory
	db, err := maxminddb.Open(getGeoLite2CityPath())
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Connect to MongoDB
	sess, err := mgo.Dial(mongoURL())
	if err != nil {
		log.Fatal(err)
	}
	defer sess.Close()

	sess.SetMode(mgo.Monotonic, true)

	// Set the namespace of the visits collection
	visitsNamespace = sess.DB("").Name + ".visits"

	// Setup the event source
	es := eventsource.New(nil, nil)
	defer es.Close()

	http.Handle("/events", es)
	http.Handle("/", http.FileServer(http.Dir(".")))

	go func() {
		// Tail the OpLog
		ops, errs := gtm.Tail(sess, &gtm.Options{nil, NewVisits})

		// Tail returns 2 channels - one for events and one for errors
		for {
			// loop forever receiving events
			select {
			case err = <-errs:
				// handle errors
				log.Println(err)
			case op := <-ops:
				if ipStr, ok := op.Data["ip"]; ok {
					// Parse the IP
					ip := net.ParseIP(ipStr.(string))
					id := op.Data["_id"].(bson.ObjectId).Hex()

					var v visitEvent
					err = db.Lookup(ip, &v)
					if err != nil {
						log.Fatal(err)
					}

					if v.HasEnglishCityName() {
						if v.InScandinavia() {
							es.SendEventMessage(v.JSON(id), "visit", id)
							fmt.Print(".")
						} else {
							fmt.Print("\n")
							log.Println(v.JSON(id))
						}
					}
				}
			}
		}
	}()

	addr := getAddr()
	log.Printf("Starting listening on http://0.0.0.0%s/", addr)
	log.Fatal(http.ListenAndServe(getAddr(), nil))
}

func getGeoLite2CityPath() string {
	if path := os.Getenv("GEOLITE2_CITY_PATH"); path != "" {
		return path
	}

	return "data/GeoLite2-City.mmdb"
}

func getAddr() string {
	if port := os.Getenv("PORT"); port != "" {
		return ":" + port
	}

	return ":6600"
}

func mongoURL() (url string) {
	url = os.Getenv("MONGOHQ_URL")

	if url == "" {
		log.Println("ENV variable MONGOHQ_URL not set!")
		os.Exit(1)
	}

	return
}
