# Explanation of aggregations

## Data Sources:

- Condor CPU Efficiency
    - Source: [HTCondor Job Monitoring](https://cmsmonit-docs.web.cern.ch/cms-htcondor-es/)
    - Fields: [ClassAds and calculated fields](https://cmsmonit-docs.web.cern.ch/cms-htcondor-es/cms-htcondor-es/)
    - MONIT OpenSearch: monit-opensearch.cern.ch `monit_prod_condor_raw_metrics`
    - HDFS: `/project/monitoring/archive/condor/raw/metric`
- StepChain CPU efficiency
    - Source and fields: [dmwm/WMArchive](https://github.com/dmwm/WMArchive)
    - MONIT OpenSearch: monit-opensearch `monit_prod_wmarchive`
    - HDFS: `/project/monitoring/archive/wmarchive/raw/metric`

## Condor CPU efficiency calculations

- Filter: `Status='Completed' AND JobFailed=0`
- [Condor Spark Job](../spark/cpueff_condor_goweb.py)

Please see used ClasAds
in [ClassAds and calculated fields](https://cmsmonit-docs.web.cern.ch/cms-htcondor-es/cms-htcondor-es/) for their
details.

| FIELD                   | EXPLANATION                                                                                                                                                      |
|-------------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Type**                | group-by tag. value: (analysis, production, tier0, test)                                                                                                         |
| **Workflow**            | group-by tag. value: Workflow name                                                                                                                               |
| **WmagentRequestName**  | group-by tag. value: WMAgent_RequestName                                                                                                                         |
| **Site**                | group-by tag. value: Site [Used only in details row calculations]                                                                                                |
| **CpuEff**              | agg field. value: `sum(CpuTimeHr)/sum(CoreTimeHr) * 100`                                                                                                         |
| **CpuEffOutlier**       | safety filter. if CpuEff > 100%, it is set as 1. Mostly it happens if "CoreTimeHr" is so low. It is a decided field by the HTCondor Job Monitoring stakeholders. |
| **Cpus**                | agg field. value: `sum(RequestCpus)`                                                                                                                             |
| **CpuTimeHr**           | agg field. value: "CpuTimeHr" is directly from its original value, represents `sum(CpuTimeHr)`                                                                   |
| **WallClockHr**         | agg field. value: "WallClockHr" is directly from its original value, represents `sum(WallClockHr)`                                                               |
| **CoreTimeHr**          | agg field. value: `sum(WallClockHr * RequestCpus)`                                                                                                               |
| **WastedCpuTimeHr**     | agg field. value: `sum( (RequestCpus * WallClockHr) - CpuTimeHr)`                                                                                                |
| **CpuEffT1T2**          | agg field. value: `100 * sum(CpuTimeHr) / sum(CoreTimeHr)` **only** in T1 T2 sites.                                                                              |
| **CpusT1T2**            | agg field. value: `sum(RequestCpus)` **only** in T1 T2 sites.                                                                                                    |
| **CpuTimeHrT1T2**       | agg field. value: `sum(CpuTimeHr)` **only** in T1 T2 sites.                                                                                                      |
| **WallClockHrT1T2**     | agg field. value: `sum(WallClockHr)` **only** in T1 T2 sites.                                                                                                    |
| **CoreTimeHrT1T2**      | agg field. value: `sum(WallClockHr * RequestCpus)` **only** in T1 T2 sites.                                                                                      |
| **WastedCpuTimeHrT1T2** | agg field. value: `sum( (RequestCpus * WallClockHr) - CpuTimeHr)` **only** in T1 T2 sites.                                                                       |

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

Filter: `data.meta_data.jobstate='success' AND data.meta_data.jobtype.isin(Production,Processing,Merge,LogCollect,Harvesting)`

- [StepChain Spark Job](../spark/cpueff_stepchain_goweb.py)

Extracted fields calculations from WMArchive `data.steps.performance` object:

- raw `StepName` : step['name']
- raw `Site` : step['site']
- raw `NumOfStreams` : step['performance']['cpu']['NumberOfStreams']
- raw `NumOfThreads` : step['performance']['cpu']['NumberOfThreads']
- raw `JobCpu` : step['performance']['cpu']['TotalJobCPU']
- raw `JobTime` : step['performance']['cpu']['TotalJobTime']
- `TotalThreadsJobTime` : JobTime * NumOfThreads
- `AcquisitionEra` : collected list of step['output']['acquisitionEra']
- `EraLen` : len(AcquisitionEra)
- `StepsLen` : number of steps in `steps` json array (mostly number of cmsRun[N])

| FIELD                    | EXPLANATION                                                                                                                                                  |
|--------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Task**                 | group-by tag. value: WMCore **Task** name                                                                                                                    |
| **StepName**             | [Used only in details row calculations] group-by tag. value: WMCore job **step**'s **cmsRun[N]**                                                             |
| **JobType**              | [Used only in details row calculations] group-by tag. value: WMCore job **step**'s **jobtype**, one of Production, Processing, Merge, LogCollect, Harvesting |
| **Site**                 | [Used only in details row calculations] group-by tag. value: WMCore job **step**'s **site**                                                                  |
| **AvgCpuEff**            | agg field. value: `sum(JobCpu) / sum(TotalThreadsJobTime) * 100`                                                                                             |
| **TotalJobs**            | agg field. value: `count` of group-by Task ("and Site" in detail rows)                                                                                       |
| **NumOfSteps**           | agg field. value: `mean(NumOfSteps)`, in reality it is always same and equals to `NumOfSteps`                                                                |
| **NumOfCalculatedSteps** | agg field. value: safety check for monitoring result. Calculated `unique count of StemName` of the data                                                      |
| **NumOfThreads**         | agg field. value: "WallClockHr" is directly from its original value, represents `sum(WallClockHr)`                                                           |
| **NumOfStreams**         | agg field. value: `mean(NumOfThreads)`, in reality it is always same and equals to `NumOfThreads`                                                            |
| **AvgJobCpu**            | agg field. value: `sum( (RequestCpus * WallClockHr) - CpuTimeHr)`                                                                                            |
| **AvgJobTime**           | agg field. value: `sum(JobTime) / count` in group-by of Task ("and Site" in detail rows)                                                                     |
| **EraLength**            | agg field. value: number of AcquisitionEra, `mean(EraLen)`                                                                                                   |
| **AcquisitionEra**       | agg field. value: set of AcquisitionEra                                                                                                                      |
