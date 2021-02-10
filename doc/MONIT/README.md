# The MONIT infrastructure

The CMS MONIT data sources are:
* [ElasticSearch](https://www.tutorialspoint.com/elasticsearch/index.htm), for raw (time retention 30-40 days) and aggregated data. Can be accessed trough [Kibana](Kibana.md), [Grafana](Grafana.md), [API token](#grafana-token)
* [InfluxDB](https://www.influxdata.com/products/influxdb-overview/), can be accessed through [Grafana](Grafana.md), [API token](#grafana-token)
* [HDFS](https://www.geeksforgeeks.org/hdfs-commands/), can be accessed via spark jobs and SWAN service.

## Grafana token

Access to the Grafana data sources can be done programmatically with a [grafana proxy](http://monit-docs.web.cern.ch/monit-docs/access/monit_grafana.html) or using our [CLI tools](#cms-monitoring-cli-tools). The method can also be used to update documents as discussed in this [ticket](https://its.cern.ch/jira/browse/CMSMONIT-53). 

Remember you will need a Grafana token for authorization. Ask us if you don't have one. 

## MONIT Architecture
![MONIT architecture](MONIT.png)
