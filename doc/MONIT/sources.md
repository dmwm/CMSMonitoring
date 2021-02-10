# CMS data sources in MONIT


| Source        | Injection  | HDFS  |  ES   |
| ------------- |:-------------:| ----------:|------------:|
| CRAB          | HTTP (monit-metrics) | /project/monitoring/archive/crab | monit_prod_crab_raw_* |
| HTCondor jobs | AMQ   | /project/monitoring/archive/condor/ | monit_prod_condor_raw_metric_* |
| SSB           | Web uploader | /project/monitoring/archive/cmssst/ | monit_prod_cmssst_* |
| Task Monitoring | AMQ | | monit_prod_condor_raw_overview_, monit_prod_condor_raw_task_* |
| GlideinWMS    | AMQ | /project/monitoring/archive/glideinwms/ | monit_prod_glideinwms_* |
| SI            | AMQ | /project/monitoring/archive/cms/raw/si_condor_* | monit_prod_cms_raw_si_condor_* |
| Phedex        | AMQ | /project/monitoring/archive/phedex_dbs, /project/monitoring/archive/phedex_replicamon | monit_prod_phedex_dbs_, monit_prod_phedex_replication_* |
| Popularity    | AMQ | /project/monitoring/archive/popagg/ | monit_prod_popagg_* |
| WMAgent       | AMQ | /project/monitoring/archive/wmagent/ | monit_prod_wmagent_* |
| WMArchive     | AMQ | /project/monitoring/archive/wmarchive | monit_prod_wmarchive_* |
| XRootD (GLED) | AMQ | /project/monitoring/archive/xrootd/ | monit_prod_xrootd_enr_* |
| XRootD (AAA)  | AMQ | | monit_prod_cms_raw_aaa-test_*, monit_prod_cms_raw_aaa-ng_ |
| Rucio logs    | HTTP | | monit_prod_cms-rucio |
| Rucio events  | Hermes | | monit_prod_cms-rucio_raw_events* |
| Production and Reprocessing | AMQ |  /project/monitoring/archive/toolsandint | monit_prod_toolsandint_ |
| XCache        | AMQ | | monit_prod_cmsxcache_raw_classads, monit_prod_cmsxcache_raw_xrootd |
| FTS log analysis | AMQ | | monit_prod_cms-fts-logsanalysis_raw_metric |

## CMSWEB logs

- [monit-timber-cmsweb Read-only](https://monit-timber-cmsweb.cern.ch/kibana) managed by the e-group **es-timber-cmsweb_kibana**
- [monit-timber-cmsweb RW](https://monit-timber-cmsweb.cern.ch/kibana_rw) managed by the e-group **es-timber-cmsweb_kibana_rw**
- [monit-timber cmswebk8s](https://monit-timber.cern.ch/kibana/goto/690ddc9d47df06cd915455c1bf616b0a)

## Additional data-sources from Sqoop jobs

There are additional data sources on HDFS which are produced by CERN analytics groups or by individual requests, e.g. DBS/PhEDEx databases dumps, etc.
These sources are produced by [Sqoop jobs](https://gitlab.cern.ch/awg/awg-ETL-crons/tree/master/sqoop) maintained by CERN IT analytics/database groups, e.g.
- [job-monitoring](https://gitlab.cern.ch/awg/awg-ETL-crons/blob/master/sqoop/cms-jm.sh)
- [jm-data-popularity](https://gitlab.cern.ch/awg/awg-ETL-crons/blob/master/sqoop/jm-cms-data-pop.sh)
- [cmssw popularity](https://gitlab.cern.ch/awg/awg-ETL-crons/blob/master/sqoop/cmssw-popularity.sh)
- [PhEDEx block replicas](https://gitlab.cern.ch/awg/awg-ETL-crons/blob/master/sqoop/phedex-blk-replicas-snapshot.sh)
- [DBS snapshot](https://gitlab.cern.ch/awg/awg-ETL-crons/blob/master/sqoop/cms-dbs3-full-copy.sh)
- [PhEDEx file catalog](https://gitlab.cern.ch/awg/awg-ETL-crons/blob/master/sqoop/phedex-file-catalog.sh)
- [CMS ASO snapshot](https://gitlab.cern.ch/awg/awg-ETL-crons/blob/master/sqoop/cms-aso.sh)
