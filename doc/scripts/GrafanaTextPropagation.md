# Grafana TextPropagation (aka. Update Menu)

[This app](../../src/python/CMSMonitoring/grafana_textpanel_propagation.py) copies the content between two text tags in a text panel in grafana to all the matching dashboards which contains the tags. 

## Parameters

```bash
optional arguments:
  -h, --help            show this help message and exit
  --token TOKEN         Admin token
  --url URL             MONIT URL
  --source_query SOURCE_QUERY
                        query string for the source dashboard, it must have
                        json format, and contain at least one of the following
                        keys: - query - tags - uid for example
                        '{"query":"HomeTabs"}'
  --target_query TARGET_QUERY
                        query string for the target dashboards, it must have
                        json format, and contain at least one of the following
                        keys: - query - tags - uid for example
                        '{"tags":"home"}'
  --start_text START_TEXT
                        String that marks the start of the text to replace,
                        default <!--START_MENU-->
  --end_text END_TEXT   String that marks the end of the text to replace,
                        default <!--END_MENU-->
```

It requires a grafana admin token belonging to the desired organization. The token can be suplied either using the argument `--token` or by using the `GRAFANA_TOKEN` enviroment variable. 

By default the script will use the grafana-dev url. For example to update the menu on the home pages, you should update it in the HomeTabs dashboards and then run:

```bash
python ~/CMSMonitoring/src/python/CMSMonitoring/grafana_textpanel_propagation.py --url "https://monit-grafana.cern.ch"
```

The defaults will search for the dashboards with the `home` tag, and update them if the menu content is different from the one in HomeTabs.