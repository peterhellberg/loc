# Loc

[SSE](http://en.wikipedia.org/wiki/Server-sent_events) stream of visitor locations plotted on a map using [D3.js](http://d3js.org/).

![Loc example map](http://assets.c7.se/skitch/loc_-_plot_visits_using_gtm_maxminddb-golang_eventsource_and_d3js-20140705-214438.png)

## Requirements

 - A MongoDB user with oplog access
 - A `visits` collection with documents including the field `ip`
 - The free GeoLite2 City database from [MaxMind](http://dev.maxmind.com/geoip/geoip2/geolite2/)

## Installation

```
go get github.com/peterhellberg/loc
```

## Environment variables

 - **PORT**
 - **GEOLITE2_CITY_PATH**
 - **MONGOHQ_URL**
