# Grafana dashboard exporter cron job

Docker image for the grafana-dashboard-exporter Kubernetes CronJob. Exports CMS Monitoring Grafana dashboard JSON definitions, compresses them, and writes the archive to EOS at `/eos/cms/store/group/offcomp_monit/grafana_backup`.

Built on `cmsmon-py` (`helpers/otel_setup.py`). Set `OTEL_ENABLED=true` and `OTEL_SERVICE_NAME` in the Kubernetes cronjob to export logs to the OpenTelemetry Collector.

## Build and publish

```sh
docker build -t registry.cern.ch/cmsmonitoring/grafana-dashboard-exporter:test .
docker push registry.cern.ch/cmsmonitoring/grafana-dashboard-exporter:test
```

## Kubernetes deployment

CronJob definition: `cmsmon-opentofu/modules/kubernetes/submodules/cronjobs/cron_grafana_dashboard_exporter.tf`
