# Configuration

This is the most important file for our intelligence module. The intelligence is controlled from here. Let's see what each field mean and how important they are. 

*The given config file format below should be followed.*

*Same config file can be used for production run as well as testing purpose. For testing purpose extra config is required which are given a "+" mark beside them.*

- Table of contents
  * [Parts](#parts)
    - [Server](#server)
    - [Annotation Dashboard](#annotation-dashboard)
    - [Alerts](#alerts)
    - [Silence](#silence)
    - [Services](#services)
  * [Config File](#config-file-example)

# Parts

The config file consists of mainly four parts :-

- Server
- Annotation Dashboard
- Alerts
- Silence
- Services

# Server

Contains all the values related to servers (Alertmanager, alerting services).

- **cmsmonURL** - CMS Monitoring Infrastructures URL. 
- **getAlertsAPI** -  AlertManager API endpoint for fetching alerts (GET Request).   
- **getSuppressedAlertsAPI** - AlertManager API endpoint for fetching suppressed alerts (GET Request). 
- **getSilencesAPI** - AlertManager API endpoint for fetching silences (GET Request). 
- **postAlertsAPI** - AlertManager API endpoint for creating alerts (POST Request). 
- **postSilenceAPI** - AlertManager API endpoint for creating silences (POST Request). 
- **deleteSilenceAPI** - AlertManager API endpoint for deleting silences (DELETE Request). 
- **httpTimeout** - HTTP timeout for all requests (in sec).
- **interval** - Time interval at which intelligence module repeats itself (in sec).
- **verbose** - verbosity level for better debugging 
    - 0 - no verbosity
    - 1 - first level of verbosity
    - 2 - second level of verbosity
    - 3 - deep level verbosity
- **dryRun** - boolean flag for dry run, if true intelligence module will not make any changes in Alermanager (doesn't create alerts, silences etc.) and doesn't annotate Grafana Dashboard. Use for debbuging purpose.
- **+** **testing** - storing details about test scenario
    - **testfile** - test cases file name for testing (Fake alerts).
    - **lifetimeOfTestAlerts** - lifetime of test alerts (fake alerts). (in minutes)
    - **annotateTestStatus** - boolean value which becomes true if dashboards annotation is successful during testing else remains false.

**Fields with + mark are for testing purpose.**

##### Defaults

- **cmsmonURL** - https://cms-monitoring.cern.ch
- **getAlertsAPI** -  /api/v1/alerts?active=true&silenced=false&inhibited=false&unprocessed=false
- **getSuppressedAlertsAPI** - /api/v1/alerts?active=false&silenced=true
- **getSilencesAPI** - /api/v1/silences
- **postAlertsAPI** - /api/v1/alerts 
- **postSilenceAPI** - /api/v1/silences
- **deleteSilenceAPI** - /api/v1/silence
- **httpTimeout** - 3
- **interval** - 1
- **verbose** - 0 
- **dryRun** - false
- **+** **testing** - storing details about test scenario
    - **testfile** - /tmp/test_cases.json
    - **lifetimeOfTestAlerts** - 5
    - **annotateTestStatus** - false

# Annotation Dashboard

Contains all the values required for annotation of Grafana Dashboards.

- **url** - Grafana Dashboard base URL.
- **dashboardSearchAPI** - Grafana API endpoint for searching dashboards with given list of tags.
- **annotationAPI** - Grafana API endpoint for creating annotations.
- **\* tags** - List of tags for the dashboards where to create annotations.
- **\* token**- Grafana Proxy Token, required for creating a annotation request.
- **dashboardsCacheExpiration** - Dashboard Cache Expiration (in hours). Intelligence module runs infinitely as a service in background. To increase it's latency and make less request to Grafana dashboards, a cache is maintained which contains all dashboards value and upon expiration the cache gets updated.

**Fields with * mark are required.**

##### Defaults

- **url** - https://monit-grafana.cern.ch
- **dashboardSearchAPI** - /api/search
- **annotationAPI** - /api/annotations
- **dashboardsCacheExpiration** - 1

# Alerts

- **uniqueLabel** - Label which defines an alert uniquely required for processing a specific alert.
- **severityLabel** - Label for severity level of an alert required while assigning proper severity level.
- **serviceLabel** - Label for service of an alert.
- **\* severityLevels** - map for defined severity levels and their priority.
- **defaultSeverityLevel** - Default severity level value in case intelligence module is not able to assign one.

**Fields with * mark are required.**

##### Defaults

- **uniqueLabel** - alertname
- **severityLabel** - severity
- **serviceLabel** - service
- **defaultSeverityLevel** - info

# Silence

- **createdBy** - Name of the creater of silences.
- **comment** -  Comment for creating silences
- **\* silenceStatus** - Labels for status of the silence. 
    - ["active", "expired", "pending"] this should be fixed unless there's change in AlertManager. DO NOT CHANGE THE VALUES OR THEIR ORDER.

**Fields with * mark are required.**

##### Defaults

- **createdBy** - admin
- **comment** -  silence by intelligence module
- **\* silenceStatus** - ["active", "expired", "pending"]

# Services

- **name** - Name of a service (eg. SSB, GGUS)
- **# keywordLabel** - Field in which the intelligence module tries to match keywords.
- **# defaultLevel** - Default Severity Level assigned to the alert at the time of it's creation by alerting services (ggus_alerting, ssb_alerting).
- **severityMap** - Map for severity levels for a service.
- **annotationMap** - Map for Dashboard annotations' keywords.
    - **# label** - Field in which the intelligence module tries to match keywords for the following actions and systems.
    - **actions** - List of actions (eg. outage, maintenance, intervention)
    - **systems** - List of services which are involved (eg. Network, Database, rucio etc.)

**Fields with * mark are required. Fields with # mark should not be changed unless there's any change in the codebase for the same.**

- **keywordLabel for SSB should be "shortDescription" and for GGUS "Priority"**.
- **defaultLevels for SSB should be "notification" and for GGUS "ticket"**.
- **label for annotationMap for SSB should be "shortDescription" and for GGUS "Subject"**.

DO NOT MAKE CHANGES FOR SSB AND GGUS SERVICES UNLESS YOU MAKE CHANGES ACCORDINGLY IN GGUS & SSB ALERTING SERVICES. 
YOU ARE FREE TO SET VALUES FOR NEW SERVICES THOUGH.

# Config File Example

```json
{
  "server": {
    "cmsmonURL": "https://cms-monitoring.cern.ch", 
    "getAlertsAPI": "/api/v1/alerts?active=true&silenced=false&inhibited=false&unprocessed=false",
    "getSuppressedAlertsAPI": "/api/v1/alerts?active=false&silenced=true",
    "getSilencesAPI": "/api/v1/silences",
    "postAlertsAPI": "/api/v1/alerts",
    "postSilenceAPI": "/api/v1/silences",
    "deleteSilenceAPI": "/api/v1/silence",
    "httpTimeout": 3,
    "interval": 1,
    "verbose": 0,
    "dryRun": false,
    "testing": {
      + "testfile": "/tmp/test_cases.json",
      "lifetimeOfTestAlerts": 5,
      "annotateTestStatus" : false
    }
  },

  "annotationDashboard": {
    "url": "https://monit-grafana.cern.ch",
    "dashboardSearchAPI": "/api/search",
    "annotationAPI": "/api/annotations",
    * "tags": ["cmsweb-play"],
    * "token": "",
    "dashboardsCacheExpiration": 1
  },

  "alerts": {
    "uniqueLabel": "alertname",
    "severityLabel": "severity",
    "serviceLabel": "service",
    * "severityLevels": {
      "info": 0,
      "warning": 1,
      "medium": 2,
      "high": 3,
      "urgent": 4
    },
    "defaultSeverityLevel": "info"
  },

  "silence": {
    "createdBy": "admin",
    "comment": "maintenance",
    #* "silenceStatus": ["active", "expired", "pending"]     #DO NOT CHANGE IT
  },

  * "services": [
    {
      "name": "SSB",
      # "keywordLabel": "shortDescription",         #DO NOT CHANGE IT, UNIQUE FOR SSB
      # "defaultLevel": "notification",               #DO NOT CHANGE IT, UNIQUE FOR SSB
      "severityMap": {                   
        "update": "info",
        "configuration": "info",
        "support": "info",
        "patching": "info",
        "upgrade": "warning",
        "intervention": "warning",
        "migration": "warning",
        "interruption": "medium",
        "risk": "high",
        "down": "urgent"
      },
      "annotationMap": {
        # "label": "shortDescription",              #DO NOT CHANGE IT, UNIQUE FOR SSB
        "actions": ["intervention"],
        "systems": ["network", "database", "db"]
      }
    },

    * {
      "name": "GGUS",                         
      # "keywordLabel": "Priority",                #DO NOT CHANGE IT, UNIQUE FOR GGUS
      # "defaultLevel": "ticket",                  #DO NOT CHANGE IT, UNIQUE FOR GGUS
      "severityMap": {                   
        "less urgent": "medium",
        "urgent": "high",
        "very urgent": "urgent"
      },
      "annotationMap": {
        # "label": "Subject",              #DO NOT CHANGE IT, UNIQUE FOR GGUS
        "actions": ["down", "failure", "outage"],
        "systems": ["network", "rucio"]
      }
    }
  ]
}
```
