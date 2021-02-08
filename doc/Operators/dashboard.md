# Dashboard maineinance

### CMS code repositories

- [CMSMonitoring](https://github.com/dmwm/CMSMonitoring) repository contains codebase used across CMS tools for monitoring purposes.
In particular, we rely on common [StompAMQ](https://github.com/dmwm/CMSMonitoring/blob/master/src/python/CMSMonitoring/StompAMQ.py)
layer to inject our data into MONIT infrastructure. All data we send to MONIT should be injected via StompAMQ module.
The CMSMonitoring repository also provides common data validation layer.

- [data_audit](https://github.com/vkuznet/data_audit) repository contains codebase and instructions to generate
CMS data usage in MONIT.

- [json_backup](https://github.com/vkuznet/json_backup) repository contains code to back all CMS grafana dashboards

### Kibana dashboards and instances

- There are several Kibana instances available to CMS users, with different access and time retention policies, as described [here](https://monit-grafana.cern.ch/d/iduu-UgZz/cms-kibana?orgId=11)
  - to save persistent plots you need to use the appropriate RW kibana instance
  - you need to be in the appropriate e-group 

### Grafana dashboards

- Grafana dashboards are organised in Production, Development, and Playground folders. The authorization schema is:
  - Playground: everybody has read/write rights. This is meant for everybody to use and experiment with graphana 
  - Development: the "CMS users" grafana team has editor rights. These are dashboards under development for specific groups, which are meant to become Production dashboards 
  - Production: these dashboards are managed by specific CMS grafana Teams (for example T0, SI, Jobs, CRAB, WMAgent, ...).
- Grafana Teams are groups of users belonging to CMS specific activities (T0, CRAB, etc) and managing a group of dashboards. 
To see all Teams and Team members consult the [Contact](https://monit-grafana.cern.ch/d/cU_crlhik/cms-monitoring-contacts?panelId=8&orgId=11") section.
- The CMS monitoring team (currently Valentin, Federica and the CatA operator) has administration rights in grafana and is responsible for:
  - create new grafana teams and/or add new members to the team, and keep up to date the corresponding [Contact](https://monit-grafana.cern.ch/d/cU_crlhik/cms-monitoring-contacts?panelId=8&orgId=11") section;
  - create new grafana data sources 
  - create API keys upon request
  - All of these features are available under the Configuration menu in the left tab. Please refer to the grafana documentation for details.


