# CMS Monitoring infrastructure

## Architecture

It consists of a few components:
- the CMS MONIT infrastructure which provides
  [ElasticSearch](https://www.tutorialspoint.com/elasticsearch/index.htm),
  [InfluxDB](https://www.influxdata.com/products/influxdb-overview/),
  [HDFS](https://www.geeksforgeeks.org/hdfs-commands/).
  See the specific [documentation](../MONIT/README.md).

- the CMS Monitoring infrastructure which provides
  [Prometheus](https://prometheus.io/),
  [AlertManager](https://www.prometheus.io/docs/alerting/latest/alertmanager/),
  [VictoriaMetrics](https://github.com/VictoriaMetrics/VictoriaMetrics)
  and other services.
- the [NATS](https://nats.io/) cluster for real-time monitoring: [documentation](https://github.com/dmwm/CMSMonitoring/blob/master/doc/NATS/nats.md).

A good overview can be found in this [talk](https://indico.cern.ch/event/873410/contributions/3682300/attachments/1966507/3270012/CMS_monitoring_infrastructure.pdf). 
You can view how all pieces are interconnected in the architectural [diagram](#architectural-diagram).

## CMS Prometheus services
We use [Prometheus](https://github.com/dmwm/CMSMonitoring/tree/master/doc/Prometheus) to monitor CMS nodes, and services.
It provides [PromQL](https://prometheus.io/docs/prometheus/latest/querying/basics/)
(Prometheus Query Language) to query your data which is accessible from
[cms-monitoring.cern.ch](https://cms-monitoring.cern.ch). In our infrastructure
we use [VictoriaMetrics](https://github.com/VictoriaMetrics/VictoriaMetrics)
back-end for Prometheus. It provides a [MetricsQL](https://victoriametrics.github.io/MetricsQL.html)
which extends capability of PromQL even further.

### Access to Prometheus and VictoriaMetrics via dashboards

Data from Prometheus/VictoriaMetrics can be visualised in [Grafana](https://github.com/dmwm/CMSMonitoring/doc/MONIT/Grafana.md) dashboards by using the appropriate data sources,
or indirectly via [promxy](https://github.com/jacksontj/promxy) proxy
service. In promxy data-source you can use either PromQL or MetricsQL in Grafana dashboards.
We gradually migrate our infratructure to only rely on
[promxy](https://github.com/jacksontj/promxy) proxy service for access to
dashboards maintaned by Prometheus or VictoriaMetrics services.

[Instructions](https://gist.github.com/vkuznet/4ba2d063cd68fb2b912e4514b05a7081) to setup terminal based dashboard to monitor remote node with Prometheus.


## CMS Monitoring CLI tools
All CMS Monitoring tools are accessible from `/cvmfs/cms.cern.ch/cmsmon` area and in 
our [github](https://github.com/dmwm/CMSMonitoring) repository. 
The CMS monitoring tools are written in python and in [Go](https://indico.cern.ch/event/912571/contributions/3837964/note/).

They include:
- `monit` allows access to CERN MONIT data-sources like ElasticSearch and InfluxDB. Just type "bin/monit --help" for instructions on how to use it. 
- `promtool` allows to access Prometheus service
- `amtool` allows to access AlertManager service
- `annotationManager` allows to manage annotations in dashboards
- `ggus_parser` allows to parse GGUS tickets
- `ssb_parser` allows to parse SSB tickets
- `hey` tools can be used to prove HTTP services via scalable, concurrent HTTP requests
- `prometheus` is Prometheus server
- `stern` is a tool to view kubernetes logs
- `nats-pub` and `nats-sub` are tools to connect to NATS server

## Architectural diagram

![cluster architecture](CMSMonitoringHA.png)

