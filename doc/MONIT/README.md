# The MONIT infrastructure

The CMS MONIT data sources are:
* [ElasticSearch](https://www.tutorialspoint.com/elasticsearch/index.htm), for raw (time retention 30-40 days) and aggregated data. Can be accessed trough [Kibana](Kibana.md), [Grafana](Grafana.md), [API token](#grafana-token)
* [InfluxDB](https://www.influxdata.com/products/influxdb-overview/), can be accessed through [Grafana](Grafana.md), [API token](#grafana-token)
* [HDFS](https://www.geeksforgeeks.org/hdfs-commands/), can be accessed via spark jobs and SWAN service.

## MONIT Architecture
![MONIT architecture](MONIT.png)
