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
### The Logic behind intelligence module
1) The main function will run all pipeline's logic.
2) pipeline starts with fetching alerts,
3) then to process each alert seeing if it has already been processed and silenced. If they are then we ignore that alert, otherwise, we pass it to the next pipeline component. ( So here we require the SilencedMap which stores all those alerts which are in Silence Mode which indicates they are processed and we should not repeat the intelligence process again for them ).
4) Then the alert comes to keyword matching were keywords are matched and accordingly severity is assigned.
5) Passes through ML Box with no logic as of now.
6) then the processed alert is pushed to AM and,
7) Then the old alert with default severity is silenced
8) Then Some resolved alerts that are silenced (i.e. when GGUS alerts are resolved) are deleted.

Regarding Counters and Testing:-
When we are fetching the alerts at step 2, we will count the number of alerts in the AM (i.e. before intelligence module does its stuff)
Then when we go to preprocessing step 3, we will count all active, expired silences when we update our SilenceMap
When we push Alerts we count how many alerts got pushed.
When we create new Silences we will add 1 to the Active Silence counter which we modified at step 3
When we delete a resolved alert's Silence we will add 1 the Expired Silence counter which we modified at step 3.

Now at the end of pipeline. We will end up having following counters:-

No Of Alerts
No Of Active Silences
No Of Expired Silences
No of Alerts Pushed

Now to verify if everything went well at the end of pipeline. We will again fetch Alerts/Silences from AM and count them and check if they matches with the counters above. If they match go to next iteration otherwise stop the testing.