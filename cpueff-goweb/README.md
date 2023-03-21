## cpueff-goweb package

[![Go Report Card](https://goreportcard.com/badge/github.com/dmwm/CMSMonitoring/cpueff-goweb)](https://goreportcard.com/report/github.com/dmwm/CMSMonitoring/cpueff-goweb)
[![GoDoc](https://img.shields.io/static/v1?label=godoc&message=reference&color=blue)](https://pkg.go.dev/github.com/dmwm/CMSMonitoring/cpueff-goweb)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](https://opensource.org/licenses/MIT)

The `cpueff-goweb` package serves Condor Job Monitoring and WMArchive stepchain CPU efficiency aggregated data, which is
stored in MongoDB,
using JQuery [datatables](https://datatables.net/).

###### Used software

* [DataTables](https://datatables.net/): very popular JQuery library to show pretty tables with nice UI and search/query
  functionalities
* [MongoDB](https://www.mongodb.com/): used to store following data in separate collections. Multiple MongoDB indexes
  are created to use full
  performance of it.
    * Condor Job monitoring aggregated results,
    * short url `hashId:request` binding
    * data source timestamp
* JQuery/JS: to manipulate and customize DataTables, JQuery and JS used
* Go [gin-gonic](https://github.com/gin-gonic) web framework is used to serve web pages and MongoDB query API calls

###### Main page functionalities

- Sort
- Detailed workflows: green "+" button
- Paging
- Count of search result
- Search using SearchBuilder conditions: "Add condition". Even though SB allows nested conditions, now it supports
  depth=1
- Buttons:copy, excel, PDF, column visibility
- Short URL: which is the advanced functionality of this service. Please see its documentation for more details.

### Docs

- [Calculations](docs/aggregations.md)
- [Short URL implementation](docs/short_url.md)

### Pipeline

![alt text](docs/pipeline.png "data pipeline")

###### Bug Report & Contribution

Please open [GitHub issue](https://github.com/dmwm/CMSMonitoring/issues)

###### References

- https://github.com/dmwm/das2go
- https://dev.to/hackmamba/build-a-rest-api-with-golang-and-mongodb-gin-gonic-version-269m
- https://github.com/gin-gonic/gin/issues/346
