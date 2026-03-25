# Promxy to Opensearch

## Purpose

This is the codebase for the `promxy-to-opensearch` docker image inside the cmsmonitoring repository of the CERN docker registry.

This image's main purpose is to serve as a flexible base for creating Kubernetes cronjobs that handle data pipelines between Promxy and OpenSearch.

## Deployment

The code for deploying the cronjob in charge of the data loads is in the [CMSKubernetes repository](https://github.com/dmwm/CMSKubernetes/blob/master/kubernetes/monitoring/crons/promxy-to-opensearch.yaml).
[Comment] # TODO Update the readme when the cronjob is moved to terraform

To deploy or update the cronjob using the yaml, you have to first set the proper `kubectl` context and then apply the yaml with the changes you made by running the following:

``` bash
export KUBECONFIG=<path_to_kubeconfig>
kubectl apply -f <path_to_cronjob_yaml>
```

The most common update to the cronjob will be changing the data it pushes to Opensearch. This can be done by changing the PromQL query passed through the cronjob environment variables. The full list of the environment variables with their respective usage is:

- PROMXY_URL: Url to use as the Promxy endpoint. Can be modified to choose between different Promxy instances or to switch between using the `query` or the `query_range` parameter (more info in the [official documentation](https://prometheus.io/docs/prometheus/latest/querying/basics/)).The default value for this variable is `"http://cms-monitoring.cern.ch:30082/api/v1/query_range"`.
- PROM_QUERY: PromQL query to be used when sending the request to Promxy. It can be used to retrieve raw data from a metric or a recording rule, or as a first filter before uploading any data to a different database. Its default value is `"avg_over_time:rucio_report_used_space:1h"`
- OPENSEARCH_HOST: Opensearch instance url to where we will send the data. The default value is `"https://os-cms.cern.ch:443/os"`.
- OPENSEARCH_INDEX: Index to which we will write the data inside Opensearch.
- KRB5_CLIENT_KTNAME: Path to the keytab file to be used for Kerberos authentication in all of CERN internal services. The default value is `"/etc/secrets/keytab"`, and the file is mounted from a Kubernetes secret.
- START_DATE: Start of the time range that we want the data from. If not specified, the time range will be set to the past month.
- END_DATE: End of the time range that we want the data from. If not specified, the time range will be set to the past month.
- STEP: Time step to be used when querying Promxy, in seconds. Smaller values means more data points but might lead to data duplication, while higher values will gather less data but risk missing some data points. The optimal value depends on the granularity of the time series (i.e. 3600 for hourly series or 86400 for daily ones).
- DEBUG_MODE: If `"true"`, sets the logging level to DEBUG, increasing the verbosity of the logs.
- DRY_RUN: If `"true"`, runs the workload without uploading the data to Opensearch. Useful for debugging purposes whithout getting wrong or duplicate data inside Opensearch.

[Comment] # TODO Add setup instructions for local run.