## rucio-dataset-mon-go package

[![Go Report Card](https://goreportcard.com/badge/github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go)](https://goreportcard.com/report/github.com/dmwm/CMSMonitoring/src/go/rucio-dataset-mon-go)
[![GoDoc](https://godoc.org/github.com/dmwm/CMSMonitoring/tree/master/src/go/rucio-dataset-mon-go?status.svg)](https://godoc.org/github.com/dmwm/CMSMonitoring/tree/master/src/go/rucio-dataset-mon-go)

The `rucio-dataset-mon-go` package serves [Rucio](https://rucio.readthedocs.io/) aggregated data, imported to MongoDB,
using JQuery [datatables](https://datatables.net/).

Package development is in still progress. You may see test page from :
rocket: [here](http://cmsweb-test1-zone-b-brkegglzfmze-node-1.cern.ch:31280/) :rocket: , beware that some
functionalities not there yet.

### Docs

- [Deployment instructions](docs/Deployment.md)
- [Datatables example request](docs/example_datatables_json_request.md)

###### Bug Report & Contribution

Please open [GitHub issue](https://github.com/dmwm/CMSMonitoring/issues)

###### References

- https://github.com/dmwm/das2go
- https://dev.to/hackmamba/build-a-rest-api-with-golang-and-mongodb-gin-gonic-version-269m
- https://github.com/dmwm/CMSKubernetes/tree/master/docker/cmsmon-rucio-ds-web
- https://github.com/dmwm/CMSKubernetes/blob/master/kubernetes/monitoring/services/mongo/cmsmon-rucio-ds-web.yaml

###### Special Thanks

- Special thanks to [Valentin](https://github.com/vkuznet) for his suggestions, reviews and guidance.
- Many thanks to Danilo Piparo for bringing up the idea of this project
