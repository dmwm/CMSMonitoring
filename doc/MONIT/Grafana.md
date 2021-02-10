# Grafana

Grafana can be used to access data sources in ElasticSearch, InfluxDB, Prometheus and VictoriaMetrics.

## Documentation

* official [documentation](http://docs.grafana.org/), [Getting Started](http://docs.grafana.org/guides/getting_started/), [screencasts](http://docs.grafana.org/tutorials/screencasts/), [Tutorials](https://grafana.com/tutorials/)

## FAQ

* The dashboard I'm looking at shows nothing/weird stuff
  - check the time range you're looking at (top right) 
  - check the health of CERN central services, intervations, on the [SSB](https://cern.service-now.com/service-portal/ssb.do?area=IT)>SSB
* How are grafana dashboards organised and who can edit them?
  - Grafana dashboards are organised in Production, Development, and Playground folders. Authorization schema is:
    - Playground: everybody has read/write rights. This is meant for everybody to use and experiment with graphana 
    - Development: the "CMS users" team has editor rights. These are dashboards under development for specific groups, which are meant to become Production dashboards 
    - Production: these dashboards are managed by specific CMS Teams (such as T0, SI, Jobs, CRAB, WMAgent, ...)
  - More details can be found in this [ticket](https://its.cern.ch/jira/browse/CMSMONIT-51)
* How do I make changes to an existing grafana dashboard?
  - If it is not your dashboard, or it is a Production dashboard, please follow these steps:
    - duplicate the existing dashboard either in Playground or in Development (if you belong to the CMS user team) 
    - apply changes to the copied dashboard
    - validate the results are as expected
    - once everything is fine, you can either ask us to move it to Production, if it is going to be a new dashboard, or get in touch with the team owning the original dashboard developer (T0, CRAB, etc) to apply the changes to the original production dashboard
  - Please note that in the case of an already existing production dashboard, the changes need to be applied on that dashboard, and you cannot copy your own modified version back, to avoid link changes

* Where do I create a new grafana dashboard?
  - You can create a new dashboard in Playground (open to everybody) or in Development (you need to belong to the CMS user Team, ask us if you don't).
   Please mind the following:
    - Dashboards in Development are dashboards that are meant to be used by a group, and the final goal is to have them in Production
    - Production dashboards are managed by a specific CMS Team (T0, CRAB, etc). If you belong to one of the groups, and you're going to create dashboards for that group, please ask us to be added to that specific Team
    - If you are new to Grafana, please go through the relevant documentation in the (tutorial)(#tutorial) section

* What are the Grafana Teams?
  - Teams are groups of users belonging to CMS specific activities (T0, CRAB, etc) and managing a group of dashboards. 
  - To see all Teams and Team members consult the [Contacts](https://monit-grafana.cern.ch/d/cU_crlhik/cms-monitoring-contacts?panelId=8&orgId=11) section.

* How to use remote data sources in Grafana as they were local
  - Try out https://github.com/retzkek/grafana-proxy
