# The MONIT infrastructure

## Data injection

- Data can be injected into MONIT through AMQ or HTTP (for data producers inside the CERN network). Full set of [instructions](injection.md) for AMQ.
- CMSWEB logs are injected with [Logstash](https://github.com/dmwm/CMSMonitoring/tree/master/doc/logstash)

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
| HDFS          | logs          | 13 months   |   SWAN/analytix |

## CMS tutorials

 * Step-by-step [tutorial](https://github.com/dmwm/CMSMonitoring/blob/master/doc/training/data_injection.md) on 
    - data injection with Python, 
    - lookup with the monit [CLI tool](https://github.com/dmwm/CMSMonitoring/blob/master/doc/infrastructure/README.md#cms-monitoring-cli-tools), 
    - from lxplus for HDFS;
 * [Live](https://indico.cern.ch/event/898664/) tutorial (23.3.2020):
   - [data injection](https://github.com/dmwm/CMSMonitoring/blob/master/doc/training/tutorials/01.DataInjection.ipynb) usin SWAN notebook, 
   - [data visualization](https://github.com/dmwm/CMSMonitoring/blob/master/doc/training/tutorials/02.Visualization.md) with Grafana,
   - [Alerts](https://github.com/dmwm/CMSMonitoring/blob/master/doc/training/tutorials/03.AlertsOnGrafana.md) in Grafana.

## Documentation from MONIT

* [CERN OpenStack Cloud guide](https://clouddocs.web.cern.ch/clouddocs/)
* [CERN MONIT infrastructure](http://monit-docs.web.cern.ch/monit-docs/overview/index.html)

### MONIT Architecture
![MONIT architecture](MONIT.png)
