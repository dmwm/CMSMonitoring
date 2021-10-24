The CMS ES datasources can be generated as following
```
# obtain admin token, e.g. admin.token
# setup proper PYTHONPATH to include CMSMonitoring codebase, e.g.
export PYTHONPATH=/path/CMSMonitoring
# run script to generate datasources.json file
datasources_list.py --token <admin.token> --output datasources.json
```
