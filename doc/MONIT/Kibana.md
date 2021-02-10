# Kibana 

Kibana is used as data exploration and visualization tool for ElasticSearch data sources

## Tutorials

* how to [discover](https://www.elastic.co/guide/en/kibana/current/discover.html) your data, 
* create [plots/visualisations](https://www.elastic.co/guide/en/kibana/current/visualize.html),
* make [dashboards](https://www.elastic.co/guide/en/kibana/current/dashboard-getting-started.html).

## FAQ

* Why are there so many Kibana instances and how can I save my visualizations/dashboards?
  - There are several Kibana instances available to CMS users, with different access and time retention policies, as described [here](#the-cms-instances). 
  - To save persistent plots you need to use the appropriate RW kibana instance and you need to be in the appropriate e-group.

## The CMS instances

| Instance                                                     | Policy        | Access       | Data                         | Time retention 
| ------------------------------------------------------------ |:-------------:| ------------:|-----------------------------:|------:
| [es-cms](https://es-cms.cern.ch/kibana/app/kibana#/discover)            | R only        | cms-members  | condor jobs (only completed) | 1.5 years |
| [es-cms/rw](https://es-cms.cern.ch/kibana_rw/app/kibana#/discover)               | R/W           | cms-comp-ops | same                         |  same |
| [es-cmspublic](https://es-cmspublic.cern.ch/kibana/app/kibana#/discover)     | R only        | everybody    | condor jobs (only completed) | 1.1.2019 - 1.5.2019 |
| [es-cmspublic/rw](https://es-cmspublic.cern.ch/kibana_rw/app/kibana#/discover)    | R/W           | cms-comp-monit | same                         | same |
| [monit](https://monit-kibana.cern.ch/kibana/app/kibana#/discover)                | R/W           | everybody    | all                          | 30-40 days |
| [monit-cms](https://monit-kibana-cms.cern.ch/kibana/app/kibana#/discover)        | R only        | cms-members    | condor jobs (all) * | same |
| [monit-cms/rw](https://monit-kibana-cms.cern.ch/kibana_rw/app/kibana#/discover) ** | R/W           | cms-comp-monit | same                         |  same
| [monit-timber](https://monit-timber.cern.ch/) ***            | R/W           | cmsweb | feed from logstash                         |  7 days

\* If you would like more indices to be added, please get in touch with us (cms-comp-monit)   
\** Saved objects (searches, dashboards, visualizations) from MONIT can be exported and reimported in  MONIT-CMS (for example, from Management -> Saved objects)  
\*** This is a restricted end-point and requires explicit request to CERN MONIT team.
