## rucio-dataset-monitoring package

[![Go Report Card](https://goreportcard.com/badge/github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-monitoring)](https://goreportcard.com/report/github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-monitoring)
[![GoDoc](https://godoc.org/github.com/dmwm/CMSMonitoring/tree/master/src/go/rucio-dataset-monitoring?status.svg)](https://godoc.org/github.com/dmwm/CMSMonitoring/tree/master/src/go/rucio-dataset-monitoring)

The `rucio-dataset-monitoring` package serves [Rucio](https://rucio.readthedocs.io/) aggregated data, which is stored in MongoDB,
using JQuery [datatables](https://datatables.net/).

Package development is in still progress. You may see test page from :rocket: [here](http://cmsweb-test1.cern.ch:31280/) :rocket: , beware that some
functionalities not there yet.

### Introduction

> Rucio dataset monitoring using aggregated Spark data

Main aim of the project is to show all Rucio dataset information in a web page with required functionalities

###### Used softwares

* [DataTables](https://datatables.net/): very popular JQuery library to show pretty tables with nice UI and search/query functionalities
* [MongoDB](https://www.mongodb.com/): used to store following data in separate collections. Multiple MongoDB indexes are created to use full
  performance of it.
    * aggregated Rucio datasets results,
    * detailed dataset results,
    * short url `hashId:request` binding
    * data source timestamp
* JQuery/JS: to manipulate and customize DataTables, JQuery and JS used
* Go [gin-gonic](https://github.com/gin-gonic) web framework is used to serve web pages and MongoDB query API calls

###### Main page functionalities

- Sort
- Detailed dataset in RSEs functionality: green "+" button
- Paging
- Count of search result
- Search using SearchBuilder conditions: "Add condition". Even though SB allows nested conditions, now it supports
  depth=1
- Buttons:copy, excel, PDF, column visibility
- Short URL: which is the advanced functionality of this service. Please see its documentation for more details.

### Docs

- [Deployment instructions](docs/Deployment.md)
- [Datatables example request](docs/example_datatables_json_request.md)
- [Short URL implementation](docs/short_url.md)

###### Bug Report & Contribution

Please open [GitHub issue](https://github.com/dmwm/CMSMonitoring/issues)

###### References

- https://github.com/dmwm/das2go
- https://dev.to/hackmamba/build-a-rest-api-with-golang-and-mongodb-gin-gonic-version-269m
- https://github.com/dmwm/CMSKubernetes/tree/master/docker/cmsmon-rucio-ds-web
- https://github.com/dmwm/CMSKubernetes/blob/master/kubernetes/monitoring/services/mongo/cmsmon-rucio-ds-web.yaml
- https://github.com/gin-gonic/gin/issues/346

###### Special Thanks

- Special thanks to [Valentin](https://github.com/vkuznet) for his suggestions, reviews and guidance.
- Many thanks to Danilo Piparo for bringing up the idea of this project


