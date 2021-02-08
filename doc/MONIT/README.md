# The MONIT infrastructure

The CMS MONIT infrastructure provides:
* [ElasticSearch](https://www.tutorialspoint.com/elasticsearch/index.htm),
* [InfluxDB](https://www.influxdata.com/products/influxdb-overview/),
* [HDFS](https://www.geeksforgeeks.org/hdfs-commands/).

### CMS Monitoring dashboards
The CMS Monitoring dashboards rely on the following data-sources:
- [ElasticSearch](https://www.tutorialspoint.com/elasticsearch/index.htm)
- [InfluxDB](https://www.influxdata.com/products/influxdb-overview/)
- [Prometheus](https://prometheus.io/)
- [VictoriaMetrics](https://github.com/VictoriaMetrics/VictoriaMetrics),

Access to the ES and influxDB data sources can be done programmatically with a [grafana proxy](http://monit-docs.web.cern.ch/monit-docs/access/monit_grafana.html) or using our [CLI tools](#cms-monitoring-cli-tools). The method can also be used to update documents as discussed in this 
[ticket](https://its.cern.ch/jira/browse/CMSMONIT-53). Remember you will need a Grafana token for authorization. Ask us if you don't have one. 

