# The MONIT infrastructure

## Data sources and visualisation/access

The CMS MONIT data sources are:
* [ElasticSearch](https://www.tutorialspoint.com/elasticsearch/index.htm), can be accessed trough [Kibana](Kibana.md), [Grafana](Grafana.md), [API token](Grafana.md#grafana-token).
* [InfluxDB](https://www.influxdata.com/products/influxdb-overview/), can be accessed through [Grafana](Grafana.md), [API token](Grafana.md#grafana-token).
* [HDFS](https://www.geeksforgeeks.org/hdfs-commands/), can be accessed via spark jobs and SWAN service. Check [here](HDFS.md) for tutorials.
* List of CMS data [sources](sources.md), and [how](code.md) they are produced (and what they contain).

| Source        | Type of data  | Retention  |  Access |
| ------------- |:-------------:| ----------:|------------:|
| ES cms        | raw           | 1.5 y      | Kibana/Grafana |
| ES MONIT      | raw           | 30-40 days | Kibana/Grafana |
| ES MONIT      | aggregated    | infinite | Kibana/Grafana |
| InfluxDB      | aggregated    | infinite   |   Grafana |
| HDFS          | raw           | infinite   |   SWAN/analytix |
| HDFS          | logs           | 13 months   |   SWAN/analytix |

## Data injection

[Tutorial](injection.md) on how to inject data in MONIT though AMQ-

## Documentation from MONIT

* [CERN OpenStack Cloud guide](https://clouddocs.web.cern.ch/clouddocs/)
* [CERN MONIT infrastructure](http://monit-docs.web.cern.ch/monit-docs/overview/index.html)

## CMS tutorials

 * [Live](https://indico.cern.ch/event/898664/) tutorial (23.3.2020), covers data injection, data visualization from ElasticSearch and access to HDSF

## MONIT Architecture
![MONIT architecture](MONIT.png)
