# Openstack Accounting cron job

Code base of the Docker image used for the openstack-accounting cronjob, which generates data about Openstack usage from different groups and publishes it in the [Openstack Quotas And Usage Monitoring website](https://cmsdatapop.web.cern.ch/cmsdatapop/eos_openstack/openstack_accounting.html) of the CMS Data Popularity service.

Built on `cmsmon-py` (`helpers/otel_setup.py`). Set `OTEL_ENABLED=true` and `OTEL_SERVICE_NAME` in the Kubernetes cronjob to export logs to the OpenTelemetry Collector.
