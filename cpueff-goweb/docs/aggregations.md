# Explanation of aggregations

## Data Sources:

- Condor CPU Efficiency
    - Source: [HTCondor Job Monitoring](https://cmsmonit-docs.web.cern.ch/cms-htcondor-es/)
    - Input Fields: [ClassAds and calculated fields](https://cmsmonit-docs.web.cern.ch/cms-htcondor-es/cms-htcondor-es/)
    - MONIT OpenSearch: monit-opensearch.cern.ch `monit_prod_condor_raw_metrics`
    - HDFS: `/project/monitoring/archive/condor/raw/metric`
- StepChain CPU efficiency
    - Source: [dmwm/WMArchive](https://github.com/dmwm/WMArchive)
    - Input
      Fields: [WMArchive performance data structure](https://github.com/dmwm/WMArchive/blob/master/doc/performance-data-structure.md)
    - MONIT OpenSearch: monit-opensearch `monit_prod_wmarchive`
    - HDFS: `/project/monitoring/archive/wmarchive/raw/metric`

## Condor CPU efficiency calculations

- Filter: `Status='Completed' AND JobFailed=0`
- [Condor Spark Job](../spark/cpueff_condor_goweb.py)

Please see used ClasAds
in [ClassAds and calculated fields](https://cmsmonit-docs.web.cern.ch/cms-htcondor-es/cms-htcondor-es/) for their
details.

| FIELD                    | EXPLANATION                                                                                                                                                                                                                                                                                 |
|--------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Type**                 | group-by tag. value: (analysis, production, tier0, test)                                                                                                                                                                                                                                    |
| **Workflow**             | group-by tag. value: Workflow name                                                                                                                                                                                                                                                          |
| **WmagentRequestName**   | group-by tag. value: WMAgent_RequestName                                                                                                                                                                                                                                                    |
| **Site**                 | group-by tag. value: Site [Used only in details row calculations]                                                                                                                                                                                                                           |
| **CoreTimeHr**           | agg field. value: `sum(WallClockHr * RequestCpus)` . [WallClockHr](https://cmsmonit-docs.web.cern.ch/cms-htcondor-es/cms-htcondor-es/#name-wallclockhr)                                                                                                                                     |
| **CpuEff**               | agg field. value: `sum(CpuTimeHr)/sum(CoreTimeHr) * 100`. [CpuTimeHr](https://cmsmonit-docs.web.cern.ch/cms-htcondor-es/cms-htcondor-es/#name-cputimehr)                                                                                                                                    |
| **CpuEffOutlier**        | safety filter. if CpuEff > 100%, it is set as 1. Mostly it happens if "CoreTimeHr" is so low. It is a decided field by the HTCondor Job Monitoring stakeholders.                                                                                                                            |
| **Cpus**                 | agg field. value: `sum(RequestCpus)`                                                                                                                                                                                                                                                        |
| **NonEvictionEff**       | agg field. value: `sum(CpuTimeHr) / sum(CommittedCoreHr)` . [CommittedCoreHr](https://cmsmonit-docs.web.cern.ch/cms-htcondor-es/cms-htcondor-es/#name-committedcorehr)                                                                                                                      |
| **EvictionAwareEffDiff** | agg field. value: `NonEvictionEff - CpuEff`                                                                                                                                                                                                                                                 |
| **ScheduleEff**          | agg field. value: `sum(CommittedWallClockHr) / sum(WallClockHr)` . [CommittedWallClockHr](https://cmsmonit-docs.web.cern.ch/cms-htcondor-es/cms-htcondor-es/#name-committedwallclockhr), [WallClockHr](https://cmsmonit-docs.web.cern.ch/cms-htcondor-es/cms-htcondor-es/#name-wallclockhr) |
| **CpuTimeHr**            | agg field. value: "CpuTimeHr" is directly from its original value, represents `sum(CpuTimeHr)` . [CpuTimeHr](https://cmsmonit-docs.web.cern.ch/cms-htcondor-es/cms-htcondor-es/#name-cputimehr)                                                                                             |
| **WallClockHr**          | agg field. value: "WallClockHr" is directly from its original value, represents `sum(WallClockHr)`. [WallClockHr](https://cmsmonit-docs.web.cern.ch/cms-htcondor-es/cms-htcondor-es/#name-wallclockhr)                                                                                      |
| **CommittedCoreHr**      | agg field. value: `sum(CommittedCoreHr)` . [CommittedCoreHr](https://cmsmonit-docs.web.cern.ch/cms-htcondor-es/cms-htcondor-es/#name-committedcorehr)                                                                                                                                       |
| **CommittedWallClockHr** | agg field. value: `sum(CommittedWallClockHr)` . [CommittedWallClockHr](https://cmsmonit-docs.web.cern.ch/cms-htcondor-es/cms-htcondor-es/#name-committedwallclockhr)                                                                                                                        |
| **WastedCpuTimeHr**      | agg field. value: `sum( (RequestCpus * WallClockHr) - CpuTimeHr)`                                                                                                                                                                                                                           |
| **CpuEffT1T2**           | agg field. value: `100 * sum(CpuTimeHr) / sum(CoreTimeHr)` **only** in T1 T2 sites.                                                                                                                                                                                                         |
| **CpusT1T2**             | agg field. value: `sum(RequestCpus)` **only** in T1 T2 sites.                                                                                                                                                                                                                               |
| **CpuTimeHrT1T2**        | agg field. value: `sum(CpuTimeHr)` **only** in T1 T2 sites.                                                                                                                                                                                                                                 |
| **WallClockHrT1T2**      | agg field. value: `sum(WallClockHr)` **only** in T1 T2 sites.                                                                                                                                                                                                                               |
| **CoreTimeHrT1T2**       | agg field. value: `sum(WallClockHr * RequestCpus)` **only** in T1 T2 sites.                                                                                                                                                                                                                 |
| **WastedCpuTimeHrT1T2**  | agg field. value: `sum( (RequestCpus * WallClockHr) - CpuTimeHr)` **only** in T1 T2 sites.                                                                                                                                                                                                  |

#### Condor Tier Efficiencies

Aggregation results over Tier and Type(analysis, production, tier0, test)

| FIELD               | EXPLANATION                                                                                        |
|---------------------|----------------------------------------------------------------------------------------------------|
| **Tier**            | group-by tag. value: (T1, T2, T3)                                                                  |
| **Type**            | group-by tag. value: (analysis, production, tier0, test)                                           |
| **TierCpuEff**      | agg field. value: `sum(CpuTimeHr)/sum(CoreTimeHr) * 100`                                           |
| **TierCpus**        | agg field. value: `sum(RequestCpus)`                                                               |
| **TierCpuTimeHr**   | agg field. value: "CpuTimeHr" is directly from its original value, represents `sum(CpuTimeHr)`     |
| **TierWallClockHr** | agg field. value: "WallClockHr" is directly from its original value, represents `sum(WallClockHr)` |

## StepChain CPU efficiency calculations

- Filter:
  `data.meta_data.jobstate='success' AND data.meta_data.jobtype.isin(Production,Processing,Merge,LogCollect,Harvesting)`
- [StepChain Spark Job](../spark/cpueff_stepchain_goweb.py)
- Input
  fields: [WMArchive performance data structure](https://github.com/dmwm/WMArchive/blob/master/doc/performance-data-structure.md)

Extracted fields calculations from WMArchive `data.steps` object:

| Intermediate Field       | Source                                                                        |
|--------------------------|-------------------------------------------------------------------------------|
| `JobType`                | `row['meta_data']['jobtype']`                                                 |
| `StepName`               | step['name'], cmsRun[N]                                                       |
| `Site`                   | step['site']                                                                  |
| `nstreams`               | `step['performance']['cpu']['NumberOfStreams']`                               |
| `nthreads`               | `step['performance']['cpu']['NumberOfThreads']`                               |
| `cpu_time`               | `step['performance']['cpu']['TotalJobCPU']` in seconds                        |
| `job_time`               | `step['performance']['cpu']['TotalJobTime']` in seconds                       |
| `acquisition_era`        | appended set of `step['output']['acquisitionEra']`                            |
| `threads_total_job_time` | `job_time * nthreads` in seconds                                              |
| `write_total_mb`         | `step['performance']['storage']['writeTotalMB']` MB size                      |
| `read_total_mb`          | `step['performance']['storage']['readTotalMB']` MB size                       |
| `peak_rss`               | `step['performance']['memory']['PeakValueRss']` (residual sum of squares) RSS |
| `peak_v_size`            | `step['performance']['memory']['PeakValueVsize']`                             |
| `era_length`             | len(acquisition_era), number of acquisitions in step list                     |
| `number_of_steps`        | counted steps in task step list                                               |

#### Final StepChain Schema

| FIELD                    | EXPLANATION                                                                                                                            |
|--------------------------|----------------------------------------------------------------------------------------------------------------------------------------|
| **Task**                 | group-by tag. value: WMCore **Task** name                                                                                              |
| **StepName**             | [Used in details] group-by tag. value: **cmsRun[N]**, WMCore job **step**'name                                                         |
| **JobType**              | [Used in details] group-by tag. value: WMCore job **step**'s **jobtype**, one of Production, Processing, Merge, LogCollect, Harvesting |
| **Site**                 | [Used in details] group-by tag. value: WMCore job **step**'s **site**                                                                  |
| **CpuEfficiency**        | agg field. value: `sum(cpu_time) / sum(threads_total_job_time) * 100`                                                                  |
| **NumberOfStep**         | agg field. value: `mean(number_of_steps)`, total number of cmsRuns of a task.                                                          |
| **MeanThread**           | agg field. value: `mean(nthreads)`, in reality it is always equal to `NumberOfThreads`                                                 |
| **MeanStream**           | agg field. value: `mean(nstreams)`, in reality it is always equal to `NumberOfStreams`                                                 |
| **MeanCpuTimeHr**        | agg field. value: `mean(cpu_time) / 3600` , average of `TotalJobCPU`(seconds) in hour unit                                             |
| **TotalCpuTimeHr**       | agg field. value: `sum(cpu_time) / 3600` , sum of `TotalJobCPU`(seconds) in hour unit                                                  |
| **MeanJobTimeHr**        | agg field. value: `mean(job_time) / 3600` , mean of `TotalJobTime`(seconds) in hour unit                                               |
| **TotalJobTimeHr**       | agg field. value: `sum(job_time) / 3600` , sum of `TotalJobTime`(seconds) in hour unit                                                 |
| **TotalThreadJobTimeHr** | agg field. value: `sum(threads_total_job_time) / 3600` , sum of `(TotalJobTime * NumberOfThreads) / 3600`(seconds) in hour unit        |
| **WriteTotalMB**         | agg field. value: `sum(write_total_mb)`                                                                                                |
| **ReadTotalMB**          | agg field. value: `sum(read_total_mb)`                                                                                                 |
| **MeanPeakRss**          | agg field. value: `mean(peak_rss)`                                                                                                     |
| **MeanPeakVSize**        | agg field. value: `mean(peak_v_size)`                                                                                                  |
| **AcquisitionEra**       | agg field. value: `acquisition_era` collected list                                                                                     |
| **EraCount**             | agg field. value: len(AcquisitionEra)                                                                                                  |
| **SiteCount**            | agg field. value: len(Sites)                                                                                                           |

## Monitoring Aggregation Pipeline

![alt text](pipeline.png "data pipeline")
