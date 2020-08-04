# CMSMonitoring/src/go/intelligence

## Overview

This intelligence Module has been developed for the CMS Infra MONIT AlertManagement. It is responsible for assigning relevant severity levels, bundling similar alerts, silencing false alerts from time to time based on it's intelligence.

## Setup

You need to set the $GOPATH at ../CMSMonitoring
```  export GOPATH=`pwd` ```
The above command will set $GOPATH to the same if your in /CMSMonitoring

## Build

This command will build the intelligence module and put the binary file in bin.
`go install go/intelligence`

## Run

This command will execute the intelligence binary. Config file path flag (-config) is mandatory. However, -verbose flag is optional.  
`intelligence -config <path-to-config-file>`

## Test
We can build the test binary residing in CMSMonitoring/src/go/intelligence/test/test.go by running
`go install go/intelligence/test`

and then we can run the below command which will test the whole pipeline by pushing test alerts, changing their severity value, annotating the Grafana dashboards and silencing unwanted alerts.
`test -config  <path-to-config-file>`


## Config File

The given config file format should be followed. The config file consists of mainly four parts :-

- Server
- Annotation Dashboard
- Alerts
- Silence
- Services

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
      "testfile": "~/CMSMonitoring/src/go/intelligence/test/test_cases.json",
      "lifetimeOfTestAlerts": 5
    }
  },

  "annotationDashboard": {
    "url": "https://monit-grafana.cern.ch",
    "dashboardSearchAPI": "/api/search",
    "annotationAPI": "/api/annotations",
    "tags": ["cmsweb-play"],
    "token": "",
    "dashboardsCacheExpiration": 1
  },

  "alerts": {
    "uniqueLabel": "alertname",
    "severityLabel": "severity",
    "serviceLabel": "service",
    "severityLevels": {
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
    "silenceStatus": ["active", "expired", "pending"]
  },

  "services": [
    {
      "name": "SSB",
      "keywordLabel": "shortDescription",
      "defaultLevel": "notification",
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
        "label": "shortDescription",
        "actions": ["intervention"],
        "systems": ["network", "database", "db"]
      }
    },

    {
      "name": "GGUS",
      "keywordLabel": "Priority",
      "defaultLevel": "ticket",
      "severityMap": {
        "less urgent": "medium",
        "urgent": "high",
        "very urgent": "urgent"
      }
    }
  ]
}
```
