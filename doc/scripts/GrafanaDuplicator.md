# Grafana duplicator

[This app](../../src/python/CMSMonitoring/grafana_duplicator.py) allows to create a copy of one (using the uid parameter) or a set of dashboards (using the query parameter) allowing to change the datasource and the id and the title. 

## Parameters

```bash
Optional arguments:
 -h, --help            show this help message and exit
  --token TOKEN         Admin token
  --url URL             MONIT URL
  --output OUTPUT       output folder for the local copy of the dashboards
  --uid DASHBOARD_UID   uid of the dashboard
  --title NEW_TITLE     if --uid is specified, this will be the title of the
                        new dashboard, if --query is used this value will be
                        used as postfix
  --datasource_replacement REPLACEMENTS REPLACEMENTS
                        pairs of values, 'orig' 'new_ds', to replace a
                        datasource in the dashboard(s). you can use several
                        values using --datasource_replacement "cmsweb-k8s"
                        "cmsweb-k8s-new" --datasource_replacement
                        "monit_es_condor_2019" "monit_es_condor"
  --query DASHBOARDS_QUERY
                        query string for the dashboards
  --store_only          Only backup the matching dashboards with the specified
                        changes, if any
```

It requires a grafana admin token belonging to the desired organization. The token can be suplied either using the argument `--token` or by using the `GRAFANA_TOKEN` enviroment variable. 

You can use either the `--uid` parameter if you want to clone a specific dashboard, or the `--query` parameter if you want to clone more than one dashboard. 

You can specify multiple datasource replacements. 

This script will duplicate the dashboards safely in grafana by default, but, if you want to prevent it, you can use the `--store_only` parameter to only store a local copy of the modified dashboards. 

