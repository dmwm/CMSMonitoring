# Code for injection to MONIT from CMS producers 

JIRA [epic](https://its.cern.ch/jira/browse/CMSMONIT-8) collecting links to code that feeds data in MONIT. 

- [WMArchive](https://github.com/dmwm/WMArchive) codebase for CMS WMArchive system
- [CMSSpark](https://github.com/dmwm/CMSSpark) framework for CMS data-source and aggregation on HDFS+Spark platform
- [CMSPopularity](https://github.com/dmwm/CMSPopularity/tree/master/ScrutinyPlot) Carl code to produce CMS popularity plots
- [cms-plots](https://github.com/dmwm/CMSPopularity/tree/master/PopularityPlot) DavidL code to produce CMS popularity plots
- [cms-htcondor-es](https://github.com/dmwm/cms-htcondor-es) ElasticSearch integration for CMS HTCondor pool
- [WMAgent monitoring](https://github.com/dmwm/WMCore/blob/master/src/python/WMComponent/AgentStatusWatcher/AgentStatusPoller.py) on ElasticSearch
- [WMStats](https://github.com/dmwm/WMCore/tree/master/src/python/WMCore/WMStats) server code
- [CRAB](https://github.com/dmwm/CRABServer/blob/master/scripts/Monitor/GenerateMONIT.py) and [ASO](https://github.com/dmwm/AsyncStageout/blob/master/bin/aso_metrics_ora.py) code to ES. 
[Aggregation](https://github.com/vkuznet/CMSSpark/blob/master/src/python/CMSSpark/aso_stats.py) from Hadoop.
- [Submission Infrastructure](https://gitlab.cern.ch/CMSSI/SubmissionInfrastructureMonitoring) monitoring scripts for the CMS global pool and multicore-specific
- Rucio [hermes module](https://github.com/rucio/rucio/blob/master/bin/rucio-hermes) and [configuration](https://github.com/dmwm/CMSKubernetes/tree/master/kubernetes/rucio)
- Production and Reprocessing [code](https://github.com/CMSCompOps/WorkflowWebTools/tree/master/workflowmonit)
- XRootD [code](https://github.com/opensciencegrid/xrootd-monitoring-collector)
