# CMSMonitoring/src/go/intelligence

## Overview

This intelligence Module has been developed for the CMS Infra MONIT AlertManagement. It is responsible for assigning relevant severity levels, bundling similar alerts, silencing false alerts from time to time based on it's intelligence.

## Environment Setup for lxplus VM Users

If you are trying to simulate the project in a lxplus VM then you will have to deploy AlertManager and ggus_alerting & ssb_alerting services as well.

#### Alertmanager Setup
Download latest version of Alertmanager. You can download it from,
https://prometheus.io/download/#alertmanager

Just copy the link of latest alertmanager binary according to your OS (here we will be using Linux one) and download it using wget util.

```$ wget "https://github.com/prometheus/alertmanager/releases/download/<version>/alertmanager-<version>.linux-amd64.tar.gz" -O am.tar.gz```

once you downloaded the alertmanager now just untar the compressed file using,

```$ tar -xzf am.tar.gz --one-top-level=am --strip-components 1```

You will see a folder named am after untar. Follow the following steps to run the Alertmanager in background as a service.

```$ cd am```
```$ nohup ./alertmanager --config.file=./alertmanager.yml </dev/null 2>&1 > AM.log &```

Now that alertmanager is up and running. Clone the CMSMonitoring repo -

```$ git clone https://github.com/dmwm/CMSMonitoring.git```

You will need to setup ggus_alerting and ssb_alerting services. They require some Environment variables to be set which you can refer from the doc [here](https://github.com/dmwm/CMSMonitoring/blob/master/doc/AlertManagement/README.md).

You need to build the binaries residing in ~/CMSMonitoring/src/go/MONIT and keep them in CMSMonitoring/bin directory.

```$ go build <file_name.go>```

Once everything is set you can run following command to run the alerting services.

```$ ggus_alert_manage start```
```$ ssb_alert_manage start```

TWO IMPORTANT THINGS TO NOTICE
- Make sure you have write permission in the directory for logging.
- Make sure to set your PATH variable to CMSMonitoring/bin, CMSMonitoring/scripts, CMSMonitoring/src/python

Now you have Alertmanager & Alerting Services running, you can now run/test the intelligence module on the Lxplus VM.

---

The above instructions are only for LXPLUX VM User. You don't need to setup Alertmanager and alerting services (GGUS/SSB) in k8s infrastructure. Just follow the below instructions.

---

## Setup

If you want the compiled binary to reside in CMSMonitoring/bin then set the $GOPATH at /CMSMonitoring by

```  export GOPATH=`pwd` ```

Then you will have to include following into the PATH variable 

```$GOPATH/bin```

You always have an option to chose any other path to $GOPATH and then build the module.

## Build

This command will build the intelligence module and put the binary file in $GOPATH/bin.

`go build go/intelligence`

## Run

This command will execute the intelligence binary. Config file path flag (-config) is mandatory. However, -verbose flag is optional.  

`intelligence -config <path-to-config-file>`

## Test

##### For Lxplux users do not deploy alerting services (GGUS & SSB). There are some fake alerts which are similar to GGUS and SSB ticketing services which are pushed into Alertmanager before starting the test. You only need to deploy AlertManager which can be done following the above instructions, though you can use Docker image for hassleless testing.

##### Binary
We can build the test binary residing in CMSMonitoring/src/go/intelligence/int_test/test.go by running

`go build go/intelligence/int_test`

and then we can run the below command which will test the whole pipeline by pushing test alerts, changing their severity value, annotating the Grafana dashboards and silencing unwanted alerts.

Test config file is residing in the /CMSMonitoring/src/go/intelligence/int_test/test_config.json.
##### CHANGE cmsmonURL to the url of Testing instance of AlertManager not the production one in test_config.json.

`test -config  <path-to-test-config-file>`

If you are not able to set environment for testing in lxplux VM, all test scenario for lxplux user has been automated using the [test_wrapper.sh](https://github.com/dmwm/CMSMonitoring/blob/master/src/go/intelligence/int_test/test_wrapper.sh) script residing in CMSMonitoring/src/go/intelligence/int_test/test_wrapper.sh. You just need to clone the repository at some directory, set the PATH variable and run [test_wrapper.sh](https://github.com/dmwm/CMSMonitoring/blob/master/src/go/intelligence/int_test/test_wrapper.sh)

```$ git clone https://github.com/dmwm/CMSMonitoring.git```

```$ export PATH=/CMSMonitoring/src/go/intelligence/int_test/:$PATH```

Set the "testfile" : <YOUR WORKING DIRECTORY TOP>/test_cases.json in [test_config.json](https://github.com/dmwm/CMSMonitoring/blob/master/src/go/intelligence/int_test/test_config.json)
For instance if you are doing test in "/tmp/<your-name>", set the "testfile" field to "/tmp/test_cases.json".

Now finally run the below command to start testing.

```$ test_wrapper.sh /tmp/<your-name>``` 



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
      "testfile": "/tmp/test_cases.json",
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