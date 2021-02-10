# CMS data sources in MONIT


| Source        | Injection  | HDFS  |  ES   |
| ------------- |:-------------:| ----------:|------------:|
| CRAB          | HTTP (monit-metrics) | /project/monitoring/archive/crab | [monit_prod_crab_raw](https://monit-kibana.cern.ch/kibana/goto/4aca05357c3b9b1863cb48a61ac6c05d) |
| HTCondor jobs | AMQ   | /project/monitoring/archive/condor/ | [monit_prod_condor_raw_metric](https://monit-kibana.cern.ch/kibana/goto/f481289f8b1c9b5a521809ebb1c73ec9) |
| SSB           | Web uploader | /project/monitoring/archive/cmssst/ | monit_prod_cmssst_* |
| Task Monitoring | AMQ | | [monit_prod_condor_raw_overview](https://monit-kibana.cern.ch/kibana/goto/16641938bdf0e757eb600deff8ec4f58), [monit_prod_condor_raw_task](https://monit-kibana.cern.ch/kibana/goto/e85d02a2411a1646fd33b9973273b7d6) |
| GlideinWMS    | AMQ | /project/monitoring/archive/glideinwms/ | [monit_prod_glideinwms](https://monit-kibana.cern.ch/kibana/goto/9d0192693b83e42d40e96d0182b9c3f6) |
| SI            | AMQ | /project/monitoring/archive/cms/raw/si_condor_* | [monit_prod_cms_raw_si_condor](https://monit-kibana.cern.ch/kibana/goto/d005ebd4f8efebd7eba33e294617281c) |
| Phedex        | AMQ | /project/monitoring/archive/phedex_dbs, /project/monitoring/archive/phedex_replicamon | monit_prod_phedex_dbs_, monit_prod_phedex_replication_* |
| Popularity    | AMQ | /project/monitoring/archive/popagg/ | monit_prod_popagg_* |
| WMAgent       | AMQ | /project/monitoring/archive/wmagent/ | [monit_prod_wmagent](https://monit-kibana.cern.ch/kibana/goto/ddb6ac9588fb8dd5ff3015b86c2a8106) |
| WMArchive     | AMQ | /project/monitoring/archive/wmarchive | [monit_prod_wmarchive](https://monit-kibana.cern.ch/kibana/goto/cfda40d994ab003e17bcde1d7181a2be) |
| XRootD (GLED) | AMQ | /project/monitoring/archive/xrootd/ | monit_prod_xrootd_enr_* |
| XRootD (AAA)  | AMQ | | [monit_prod_cms_raw_aaa-test](https://monit-kibana.cern.ch/kibana/goto/cfc7f48530bfcb510b6d557e632bd5ab), [monit_prod_cms_raw_aaa-ng](https://monit-kibana.cern.ch/kibana/goto/f91632d148761038fd314e909ebaeffb) |
| Rucio logs    | HTTP | | monit_prod_cms-rucio |
| Rucio events  | Hermes | | monit_prod_cms-rucio_raw_events* |
| Production and Reprocessing | AMQ |  /project/monitoring/archive/toolsandint | [monit_prod_toolsandint](https://monit-kibana.cern.ch/kibana/goto/d175ecb6b967a48697d9e5a0ab30e259) |
| XCache        | AMQ | | [monit_prod_cmsxcache_raw_classads](https://monit-kibana.cern.ch/kibana/goto/a94df5af9de3a4d8cb49c12e6cd72db7), [monit_prod_cmsxcache_raw_xrootd](https://monit-kibana.cern.ch/kibana/goto/5655e6a4ba7e2059329eca50e5beaaa2) |
| FTS log analysis | AMQ | | [monit_prod_cms-fts-logsanalysis_raw_metric](https://monit-kibana.cern.ch/kibana/goto/09ad5774e9e52e9bd862d0621d9f2d5e) |
| EOS logs         | | /project/monitoring/archive/eos-report/logs/cms | [MONIT timber](https://monit-timber.cern.ch/kibana/goto/3a94bb41d9b9627462565df8f386164c) |
| ES size | AMQ | | [](https://monit-kibana.cern.ch/kibana/goto/fc44fcd41f4093de9b4d612926984ad4) | 

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
