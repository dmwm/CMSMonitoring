# Grafana dashboard copy cron job

Docker image for the grafana-dashboard-copy Kubernetes CronJob. Copies Grafana dashboards from the "Production" folder to "Production Copy" via the Grafana API.

Built on `cmsmon-py` (`helpers/otel_setup.py`). Set `OTEL_ENABLED=true` and `OTEL_SERVICE_NAME` in the Kubernetes cronjob to export logs to the OpenTelemetry Collector.

## Build and publish

```sh
docker build -t registry.cern.ch/cmsmonitoring/grafana-dashboard-copy:test docker/grafana-dashboard-copy
docker push registry.cern.ch/cmsmonitoring/grafana-dashboard-copy:test
```

## Kubernetes deployment

CronJob definition: `cmsmon-opentofu/modules/kubernetes/submodules/cronjobs/cron_grafana_dashboard_copy.tf`
