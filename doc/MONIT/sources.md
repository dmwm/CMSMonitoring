# CMS data sources in MONIT


| Source        | Injection  | HDFS  |  ES   |
| ------------- |:-------------:| ----------:|------------:|
| CRAB          | HTTP (monit-metrics) | /project/monitoring/archive/crab | [monit_prod_crab_raw](https://monit-kibana.cern.ch/kibana/goto/4aca05357c3b9b1863cb48a61ac6c05d) |
| HTCondor jobs | AMQ   | /project/monitoring/archive/condor/ | monit_prod_condor_raw_metric_* |
| SSB           | Web uploader | /project/monitoring/archive/cmssst/ | monit_prod_cmssst_* |
| Task Monitoring | AMQ | | monit_prod_condor_raw_overview_, monit_prod_condor_raw_task_* |
| GlideinWMS    | AMQ | /project/monitoring/archive/glideinwms/ | monit_prod_glideinwms_* |
| SI            | AMQ | /project/monitoring/archive/cms/raw/si_condor_* | monit_prod_cms_raw_si_condor_* |
| Phedex        | AMQ | /project/monitoring/archive/phedex_dbs, /project/monitoring/archive/phedex_replicamon | monit_prod_phedex_dbs_, monit_prod_phedex_replication_* |
| Popularity    | AMQ | /project/monitoring/archive/popagg/ | monit_prod_popagg_* |
| WMAgent       | AMQ | /project/monitoring/archive/wmagent/ | [monit_prod_wmagent](https://monit-kibana.cern.ch/kibana/goto/ddb6ac9588fb8dd5ff3015b86c2a8106) |
| WMArchive     | AMQ | /project/monitoring/archive/wmarchive | [monit_prod_wmarchive](https://monit-kibana.cern.ch/kibana/goto/cfda40d994ab003e17bcde1d7181a2be) |
| XRootD (GLED) | AMQ | /project/monitoring/archive/xrootd/ | monit_prod_xrootd_enr_* |
| XRootD (AAA)  | AMQ | | monit_prod_cms_raw_aaa-test_*, monit_prod_cms_raw_aaa-ng_ |
| Rucio logs    | HTTP | | monit_prod_cms-rucio |
| Rucio events  | Hermes | | monit_prod_cms-rucio_raw_events* |
| Production and Reprocessing | AMQ |  /project/monitoring/archive/toolsandint | monit_prod_toolsandint_ |
| XCache        | AMQ | | monit_prod_cmsxcache_raw_classads, monit_prod_cmsxcache_raw_xrootd |
| FTS log analysis | AMQ | | monit_prod_cms-fts-logsanalysis_raw_metric |
| EOS logs         | /project/monitoring/archive/eos-report/logs/cms | |

## CMSWEB logs

- In ElasticSearch:
  - [monit-timber-cmsweb Read-only](https://monit-timber-cmsweb.cern.ch/kibana) managed by the e-group **es-timber-cmsweb_kibana**
  - [monit-timber-cmsweb RW](https://monit-timber-cmsweb.cern.ch/kibana_rw) managed by the e-group **es-timber-cmsweb_kibana_rw**
  - [monit-timber cmswebk8s](https://monit-timber.cern.ch/kibana/goto/690ddc9d47df06cd915455c1bf616b0a)
- in HDFS:  /project/monitoring/archive/cmsweb/logs, /project/monitoring/archive/cmswebk8s/logs           
                
## Additional data-sources from Sqoop jobs

There are additional data sources on HDFS which are produced by [Sqoop jobs](https://github.com/dmwm/CMSKubernetes/tree/master/docker/sqoop/scripts/) running on k8s
- [job-monitoring](https://github.com/dmwm/CMSKubernetes/tree/master/docker/sqoop/scripts/cms-jm.sh) /project/awg/cms/job-monitoring/avro-snappy
- [jm-data-popularity](https://github.com/dmwm/CMSKubernetes/tree/master/docker/sqoop/scripts/jm-cms-data-pop.sh) /project/awg/cms/jm-data-popularity/avro-snappy
- [cmssw popularity](https://github.com/dmwm/CMSKubernetes/tree/master/docker/sqoop/scripts/cmssw-popularity.sh) /project/awg/cms/cmssw-popularity/avro-snappy
- [PhEDEx block replicas](https://github.com/dmwm/CMSKubernetes/tree/master/docker/sqoop/scripts/phedex-blk-replicas-snapshot.sh) /project/awg/cms/phedex/block-replicas-snapshots
- [DBS snapshot](https://github.com/dmwm/CMSKubernetes/tree/master/docker/sqoop/scripts/cms-dbs3-full-copy.sh) /project/awg/cms/CMS_DBS3_PROD_GLOBAL/current
- [PhEDEx file catalog](https://github.com/dmwm/CMSKubernetes/tree/master/docker/sqoop/scripts/phedex-file-catalog.sh) /project/awg/cms/phedex/catalog
- [CMS ASO snapshot](https://github.com/dmwm/CMSKubernetes/tree/master/docker/sqoop/scripts/cms-aso.sh) /project/awg/cms/CMS_ASO/filetransfersdb
- [Rucio replicas](https://github.com/dmwm/CMSKubernetes/tree/master/docker/sqoop/scripts/rucio_replicas.sh) /project/awg/cms/rucio/
