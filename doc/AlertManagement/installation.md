# Installation

* [SSB Alerting Service](#ssb-alerting-service)
    + [Requirements](#requirements)
    + [Build](#build)
    + [Environment Variables](#environment-variables)
    + [Run](#run)
    + [Status](#status)
    + [Stop](#stop)
* [GGUS Alerting Service](#ggus-alerting-service)
    + [Requirements](#requirements-1)
    + [Build](#build-1)
    + [Environment Variables](#environment-variables-1)
    + [Run](#run-1)
    + [Status](#status-1)
    + [Stop](#stop-1)
* [Karma Dashboard](#karma-dashboard)
    + [Build](#build-2)
    + [Config](#config)
* [Alert CLI Tool](#alert-cli-tool)
    + [Build](#build-3)
    + [Config](#config-1)
    + [Run](#run-2)
* [Intelligence Program](#intelligence-program)
    + [Build](#build-4)
    + [Config](#config-2)
    + [Run](#run-3)
* [Intelligence Module](#intelligence-module)
* [LXPLUS VM Setup](#lxplus-vm-setup)
    + [Alertmanager Setup](#alertmanager-setup)
    + [Project Setup](#project-setup)

At first set the GOPATH to Working directory.

```export GOPATH=<WORK_DIR>```

Alerting services run on two basic programs:
- Parser - Which fetches tickets from the source
- Alerting module - converts tickets into alerts and pushes to Alertmanager

# SSB Alerting Service

LXPLUS USERS PLEASE DO NOT RUN THESE COMMANDS AT HOME DIRECTORY AS YOU DON'T HAVE WRITE PERMISSION IN ONE DIRECTORY TOP FOR ALERTING SERVICES' LOGGING. THUS, MAKE A DIRECTORY e.g. $HOME/test and then proceed. 

### Requirements

You need [Stomp](https://github.com/go-stomp/stomp) package for [monit](https://github.com/dmwm/CMSMonitoring/blob/master/src/go/MONIT/monit.go). Get the package using this command.

```go get github.com/go-stomp/stomp```


### Build

Clone the CMSMonitoring repo -

```$ git clone https://github.com/dmwm/CMSMonitoring.git```

or git pull

##### Parser 
For SSB we use [monit.go](https://github.com/dmwm/CMSMonitoring/blob/master/src/go/MONIT/monit.go).
Build the monit program by executing the command below.

```go build -o <WORK_DIR>/bin/monit <WORK_DIR>/CMSMonitoring/src/go/MONIT/monit.go```

##### Alerting Module 
For SSB we use [ssb_alerting.go](https://github.com/dmwm/CMSMonitoring/blob/master/src/go/MONIT/ssb_alerting.go).

```go build -o <WORK_DIR>/bin/ssb_alerting <WORK_DIR>/CMSMonitoring/src/go/MONIT/ssb_alerting.go```

where, WORK_DIR is your working directory. So you might want to set the PATH variable to the directory where these binaries are getting build (e.g. <WORK_DIR>/bin).

```export PATH=<WORK_DIR>/bin:$PATH```

### Environment Variables
Environments:
  - **QUERY**       -   CMS Monit ES/InfluxDB Query (eg. select * from outages where time > now() - 2h)              
  - **TOKEN**       -   Grafana Proxy user token

Environment Vars which are common for GGUS & SSB Alerting Services:-
  - **CMSMON_URL**  -   CMS Monitoring URL (default - https://cms-monitoring.cern.ch")
  - **INTERVAL**    -   Time interval at which alerting services run (in sec)  (default - 1)
  - **TIMEOUT** - HTTP client timeout operation (GGUS Parser)     default - 0 (zero means no timeout)
  - **VERBOSE**     -   Verbosity level  (default - 0)
     - 0 - no verbosity
    - 1 - first level of verbosity
    - 2 - second level of verbosity
    - 3 - deep level verbosity

### Run

We are using bash scripts for deploying the alerting services as a linux service. For this we have two scripts :-
- [ssb_alert.sh](https://github.com/dmwm/CMSMonitoring/blob/master/scripts/ssb_alert.sh) - This script automates the process of parser and alerting module.
- [ssb_alert_manage](https://github.com/dmwm/CMSMonitoring/blob/master/scripts/ssb_alert_manage) - This script gives special commands to start/stop the alerting service as a linux service.

You might want to set the PATH variable to the directory where these scripts are residing ie. (<WORK_DIR>/CMSMonitoring/scripts).

```export PATH=<WORK_DIR>/CMSMonitoring/scripts:$PATH```

Run this command to start the SSB alerting service. Make sure all environment variables are set.

```ssb_alert_manage start```

### Status

You can use status command to get the status of SSB alerting service using this command.

```ssb_alert_manage status```

### Stop

You can use stop command to stop the SSB alerting service using this command.

```ssb_alert_manage stop```


# GGUS Alerting Service

LXPLUS USERS PLEASE DO NOT RUN THESE COMMANDS AT HOME DIRECTORY AS YOU DON'T HAVE PERMISSION TO WRITE IN ONE DIRECTORY TOP FOR ALERTING SERVICES' LOGGING. THUS, MAKE A DIRECTORY e.g. $HOME/test and then proceed. 

### Requirements

You need [x509proxy](https://github.com/vkuznet/x509proxy) package for [ggus_parser](https://github.com/dmwm/CMSMonitoring/blob/master/src/go/MONIT/ggus_parser.go). Get the package using this command.

```go get github.com/vkuznet/x509proxy```

### Build

##### Parser 
For GGUS we use [ggus_parser.go](https://github.com/dmwm/CMSMonitoring/blob/master/src/go/MONIT/ggus_parser.go).
Build the ggus_parser by executing the command below.

```go build -o <WORK_DIR>/bin/ggus_parser <WORK_DIR>/CMSMonitoring/src/go/MONIT/ggus_parser.go```

##### Alerting Module 
For GGUS we use [ggus_alerting.go](https://github.com/dmwm/CMSMonitoring/blob/master/src/go/MONIT/ggus_alerting.go).

```go build -o <WORK_DIR>/bin/ggus_alerting <WORK_DIR>/CMSMonitoring/src/go/MONIT/ggus_alerting.go```

where, WORK_DIR is your working directory. So you might want to set the PATH variable to the directory where these binaries are getting build (e.g. <WORK_DIR>/bin).

```export PATH=<WORK_DIR>/bin:$PATH```

### Environment Variables
Environments:
- **GGUS_FORMAT** :   GGUS Query Format("csv" or "xml")     (default - "csv")
- **VO**          :   Required VO attribute   (default - "cms")

Environment Vars which are common for GGUS & SSB Alerting Services:-
  - **CMSMON_URL**  -   CMS Monitoring URL (default - https://cms-monitoring.cern.ch")
  - **INTERVAL**    -   Time interval at which alerting services run (in sec)  (default - 1)
  - **TIMEOUT** - HTTP client timeout operation (GGUS Parser)     default - 0 (zero means no timeout)
  - **VERBOSE**     -   Verbosity level  (default - 0)
    - 0 - no verbosity
    - 1 - first level of verbosity
    - 2 - second level of verbosity
    - 3 - deep level verbosity

### Run
We are using bash scripts for deploying the alerting services as a linux service. For this we have two scripts :-
- [ggus_alert.sh](https://github.com/dmwm/CMSMonitoring/blob/master/scripts/ggus_alert.sh) - This script automates the process of parser and alerting module.
- [ggus_alert_manage](https://github.com/dmwm/CMSMonitoring/blob/master/scripts/ggus_alert_manage) - This script gives special commands to start/stop the alerting service as a linux service.

You might want to set the PATH variable to the directory where these scripts are residing ie. (<WORK_DIR>/CMSMonitoring/scripts).

```export PATH=<WORK_DIR>/CMSMonitoring/scripts:$PATH```

Run this command to start the GGUS alerting service. Make sure all environment variables are set.

```ggus_alert_manage start```

### Status

You can use status command to get the status of ggus alerting service using this command.

```ggus_alert_manage status```

### Stop

You can use stop command to stop the ggus alerting service using this command.

```ggus_alert_manage stop```

# Karma Dashboard 

### Build

We can build the docker container using Karma [Dockerfile](https://github.com/dmwm/CMSKubernetes/blob/master/docker/karma/Dockerfile).

```docker build -t <CERN_REPO./karma <path-to-Dockerfile>```

### Config

Required Environment Variable for docker container -

```ALERTMANAGER_URI ```

Karma Dashboard can be configured using [`karma.yaml`](https://github.com/dmwm/CMSKubernetes/blob/master/docker/karma/karma.yaml) file. Below is an example of the config file used for deployment.

```json
ui:
    refresh: 30s
    hideFiltersWhenIdle: true
    colorTitlebar: false
    theme: "auto"
    minimalGroupWidth: 420
    alertsPerGroup: 6
    collapseGroups: collapsed
    multiGridLabel: "tag"
    multiGridSortReverse: false
```

k8s manifest files can be used to deploy the karma dashboard on CERN Kubernetes infrastructure.

## Alert CLI Tool

### Build

Run the following command to build [alert.go](https://github.com/dmwm/CMSMonitoring/blob/master/src/go/MONIT/alert.go).

```go build -o <WORK_DIR>/bin/alert <WORK_DIR>/CMSMonitoring/src/go/MONIT/alert.go```

where, WORK_DIR is your working directory. So you might want to set the PATH variable to the directory where the binary is going to get build (e.g. <WORK_DIR>/bin).

```export PATH=<WORK_DIR>/bin:$PATH```

### Config

We are using a json file for configuring the alert CLI tool.

- **cmsmonURL** -  CMS Monitoring URL for AlertManager (default - https://cms-monitoring.cern.ch").
- **names** - A list of labels. While detail printing of an alert, these are the main labels. You can find them in the [example](#ssb-alert-detail-printing-example) below (NAMES, LABELS, ANNOTATIONS)
- **columns** - A list of labels for the tabular printing at top.
- **attributes** - A list of attributes while detail printing for instance if you see the [example](#ssb-alert-detail-printing-example) below. There you might find "service", "severity", "tag" under "LABELS".
- **verbose** - Verbosity Level
     - 0 - no verbosity
    - 1 - first level of verbosity
    - 2 - second level of verbosity
    - 3 - deep level verbosity
- **severity** - A map for severity levels which defines each levels priority, maintained in sorted order.

An example of detail printing of an alert.
#### SSB alert detail printing example
```
NAMES: ssb-OTG0056808
LABELS
	service: SSB
	severity: info
	tag: monitoring
ANNOTATIONS
	sysModCount: 2
	sysUpdatedBy: timbell
	type: Service Change
	updateTimestamp: 2020-06-11T14:14:56Z
	feName: Monitoring
	shortDescription: MONIT HDFS Log retention policy
	ssbNumber: OTG0056808
	sysCreatedBy: sbrundu
	seName: Monitoring Service
	date: 2020-08-31T22:00:00Z
	description: IT Infrastructure Services
	monitState: OPEN
	monitState1: OPEN
```


An example of a config file.

```json
{           
    "cmsmonURL" : "https://cms-monitoring.cern.ch",
    "names": ["NAMES", "LABELS", "ANNOTATIONS"],
    "columns": ["NAME", "SERVICE", "TAG", "SEVERITY", "STARTS", "ENDS", "DURATION"],
    "attributes":["service", "tag", "severity"],
    "verbose" : 0,
    "severity" : {
            "info" : 0,
            "warning" : 1,
            "medium" : 2,
            "high" : 3,
            "critical" : 4,
            "urgent" : 5
        }
}
```

### Run

You can simply run :-

```alert```

to get all the alerts currently in the alertmanager. Screenshots for the same [here](https://github.com/dmwm/CMSMonitoring/blob/master/doc/AlertManagement/README.md#screenshots). However, it has many options to sort over various values, to filter etc. You can follow the help section below.

**FOR LXPLUS USERS**
You need to generate a config file and set the Environment Variable (CONFIG_PATH). MAKE SURE YOU SET PROPER "cmsmonURL" in config File.

```alert -generateConfig <WORK_DIR>/.alertconfig.json```

```export CONFIG_PATH=<WORK_DIR>/.alertconfig.json```


```
Usage: alert [options]
  -generateConfig
    	Flag for generating default config
  -json
    	Output in JSON format
  -name string
    	Alert Name
  -service string
    	Service Name
  -severity string
    	Severity Level of alerts
  -sort string
    	Sort data on a specific Label
  -tag string
    	Tag for alerts
  -token string
    	Authentication token to use
  -verbose int
    	verbosity level, can be overwritten in config

Environments:
	CONFIG_PATH:	 Config to use, default (/home/z3r0/.alertconfig.json)

Examples:
	Get all alerts:
	    alert

	Get all alerts in JSON format:
	    alert -json

	Get all alerts with filters (-json flag will output in JSON format if required):
	Available filters:
	service	Ex GGUS,SSB,dbs,etc.
	severity	Ex info,medium,high,urgent,etc.
	tag		Ex cmsweb,cms,monitoring,etc.

	Get all alerts of specific service/severity/tag/name. Ex GGUS/high/cms/ssb-OTG0058113:
	    alert -service=GGUS
	    alert -severity=high
	    alert -tag=cms
	    alert -name=ssb-OTG0058113

	Get all alerts based on multi filters. Ex service=GGUS, severity=high:
	    alert -service=GGUS -severity=high

	Sort alerts based on labels. The -sort flag on top of above queries will give sorted alerts.:
	Available labels:
	severity	Severity Level
	starts		Starting time of alerts
	ends		Ending time of alerts
	duration	Lifetime of alerts

	Get all alerts of service=GGUS, severity=high sorted on alert's duration:
	    alert -service=GGUS -severity=high -sort=duration

	Get all alerts of service=GGUS sorted on severity level:
	    alert -service=GGUS -sort=severity
```

# Intelligence Program

### Build

Run the following command to build [intelligence.go](https://github.com/dmwm/CMSMonitoring/blob/master/src/go/MONIT/intelligence.go).

```go build -o <WORK_DIR>/bin/intelligence <WORK_DIR>/CMSMonitoring/src/go/MONIT/intelligence.go```

where, WORK_DIR is your working directory. So you might want to set the PATH variable to the directory where the binary is going to get build (e.g. <WORK_DIR>/bin).

### Config

We are using a json file for configuring the intelligence program.

- **cmsmonURL** - CMS Monitoring Infrastructures URL. 
- **getAlertsAPI** -  AlertManager API endpoint for fetching alerts (GET Request).   
- **postSilenceAPI** - AlertManager API endpoint for creating silences (POST Request). 
- **httpTimeout** - HTTP timeout for all requests (in sec).
- **interval** - Time interval at which intelligence module repeats itself (in sec).
- **createdBy** - Name of the creater of silences.
- **severityFilter** - by convetion CMS Operators give "maintenance" in the severity level which helps us to filter the maintenance type alerts.
- **searchingLabel** -  Label for searching the instances values in the maintenance alert which are going to get affected.
- **uniqueLabel** - Label which defines an alert uniquely required for processing a specific alert.
- **comment** -  Comment for creating silences
- **verbose** - verbosity level for better debugging 
    - 0 - no verbosity
    - 1 - first level of verbosity
    - 2 - second level of verbosity
    - 3 - deep level verbosity


An example of a config file.

```json
{           
    "cmsmonURL": "http://localhost:9093",
    "getAlertsAPI": "/api/v1/alerts?active=true&silenced=false&inhibited=false&unprocessed=false",
    "postSilenceAPI": "/api/v1/silences",
    "httpTimeout": 3,
    "interval": 1,
    "createdBy": "admin",
    "severityFilter": "maintenance",
    "searchingLabel": "instance",
    "uniqueLabel": "alertname",
    "seperator": " ",
    "comment": "maintenance - false alerts getting silenced",
    "verbose": 0
}
```

### Run

You can run the intelligence program by executing the following command.

```intelligence -config <path-to-config-file>```

## Intelligence Module

Find the detailed installation guide for intelligence module [here](https://github.com/dmwm/CMSMonitoring/blob/master/src/go/intelligence/installation.md).

# LXPLUS VM Setup

### Alertmanager Setup
If you are trying to simulate the project in a lxplus VM then you will also have to deploy AlertManager.

Download the latest version of Alertmanager. You can download it from,
https://prometheus.io/download/#alertmanager

Just copy the link of latest alertmanager binary according to your OS (here we will be using Linux one) and download it using wget/curl util.

```$ wget "https://github.com/prometheus/alertmanager/releases/download/<version>/alertmanager-<version>.linux-amd64.tar.gz" -O am.tar.gz```

once you download the alertmanager, untar the compressed file using tar util.

```$ tar -xzf am.tar.gz```

Follow the following steps to run the Alertmanager in background as a service.

```$ cd alertmanager-<version>.linux-amd64.tar.gz```

```$ nohup ./alertmanager --config.file=./alertmanager.yml </dev/null 2>&1 > AM.log &```

### Project Setup
Now that alertmanager is up and running. Clone the CMSMonitoring repo -

```$ git clone https://github.com/dmwm/CMSMonitoring.git```

You will need to setup ggus_alerting and ssb_alerting services. They require some Environment variables to be set which you can refer from the doc [here](https://github.com/dmwm/CMSMonitoring/blob/master/doc/AlertManagement/README.md#ggus-alerting-service) and [here](https://github.com/dmwm/CMSMonitoring/blob/master/doc/AlertManagement/README.md#ssb-alerting-service).

Extra Environment Variables other than mentioned above which are required for setting up the project:-
- X509_USER_KEY= **PATH FOR X509_USER_KEY**
- X509_USER_CERT= **PATH FOR X509_USER_CERT**

TWO IMPORTANT THINGS TO NOTICE
- Make sure you have write permission in the directory for logging. If you don't have permission, then don't setup anything at $HOME directory. Create a directory at $HOME (eg. $HOME/test) and start setting up there only.
- Make sure to set your PATH variable to <WORK_DIR>/bin, <WORK_DIR>/CMSMonitoring/scripts.

Now you have Alertmanager & Alerting Services running. You can also run the intelligence module using the doc [here](#intelligence-module).
