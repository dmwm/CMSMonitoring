# CMSMonitoring/src/go/intelligence

## Overview
This intelligence Module has been developed for the CMS Infra MONIT AlertManagement. It is responsible for assigning relevant severity levels, bundling similar alerts, silencing false alerts from time to time based on it's intelligence.

## Setup
You need to set the $GOPATH at ../CMSMonitoring
```  export GOPATH=`pwd` ```
The above command will set $GOPATH to the same if your in /CMSMonitoring

## Build
This command will build the intelligence module and put the binary file in bin.
```go install go/intelligence ```

## Run
This command will execute the intelligence binary. Config file path flag (-config) is mandatory. However, -verbose flag is optional.  
```intelligence -config <path-to-config-file>```

## Config File
The given config file format should be followed. The config file consists of mainly four parts :-
- Server
- Alerts
- Silence
- Services

```json
{
  "server": {
    "cmsmonURL": "https://cms-monitoring.cern.ch",
    "getAlertsAPI": "/api/v1/alerts?active=true&silenced=false&inhibited=false&unprocessed=false",
    "getSilencesAPI": "/api/v1/silences",
    "postAlertsAPI": "/api/v1/alerts",
    "postSilenceAPI": "/api/v1/silences",
    "httpTimeout": 3,
    "interval": 1,
    "verbose": 0
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
    "createdBy": "intelligence module",
    "comment": "Silencing the alert - intelligence",
    "activeStatus": "active"
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
