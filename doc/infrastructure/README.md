# CMS Monitoring infrastructure

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

You can view how these pieces are interconnected in the following
architectural diagram:
![cluster architecture](CMSMonitoringHA.png)

We also provide [NATS](https://nats.io/) cluster for real-time monitoring
needs.

### CMS Prometheus services
We use [Prometheus](https://prometheus.io/) to monitor CMS nodes, and services.
It provides [PromQL](https://prometheus.io/docs/prometheus/latest/querying/basics/)
(Prometheus Query Language) to query your data which is accessible from
[cms-monitoring.cern.ch](https://cms-monitoring.cern.ch). In our infrastructure
we use [VictoriaMetrics](https://github.com/VictoriaMetrics/VictoriaMetrics)
back-end for Prometheus. It provides a [MetricsQL](https://victoriametrics.github.io/MetricsQL.html)
which extends capability of PromQL even further.

### Access to Prometheus and VictoriaMetrics via dashboards

Data from Prometheus/VictoriaMetrics can be visualised in Grafana dashboards by using the appropriate data sources,
or indirectly via [promxy](https://github.com/jacksontj/promxy) proxy
service. Therefore, in promxy data-source you can use either PromQL or MetricsQL in Grafana dashboards.
We gradually migrate our infratructure to only rely on
[promxy](https://github.com/jacksontj/promxy) proxy service for access to
dashboards maintaned by Prometheus or VictoriaMetrics services.

### CMS Monitoring CLI tools
All CMS Monitoring tools are accessible from `/cvmfs/cms.cern.ch/cmsmon` area.
They inlcude:
- `monit` allows access to CERN MONIT data-sources like
  ElasticSearch and InfluxDB.
- `promtool` allows to access Prometheus service
- `amtool` allows to access AlertManager service
- `annotationManager` allows to manage annotations in dashboards
- `ggus_parser` allows to parse GGUS tickets
- `ssb_parser` allows to parse SSB tickets
- `hey` tools can be used to prove HTTP services via scalable, concurrent HTTP requests
- `prometheus` is Prometheus server
- `stern` is a tool to view kubernetes logs
- `nats-pub` and `nats-sub` are tools to connect to NATS server

### References
For complete guide to CMS Monitoring infrastructure please refer
to our [paper](https://doi.org/10.1007/s41781-020-00051-x).
For a collection of tutorials, FAQs, and documentation about how to inject, access, and visualize data see [here](https://monit-grafana.cern.ch/d/FKrE9_2mk/cms-monitoring-tutorials?orgId=11).
