# Content

- [AlertManagement](#alertmanagement)
  * [Overview](#overview)
  * [Architectural Diagram](#architectural-diagram)
  * [Installation](#installation)
  * [SSB Alerting Service](#ssb-alerting-service)
    + [monit](#monit)
    + [ssb_alerting](#ssb_alerting)
    + [ssb_alert.sh](#ssb_alert.sh)
    + [ssb_alert_manage](#ssb_alert_manage)
  * [GGUS Alerting Service](#ggus-alerting-service)
    + [ggus_parser](#ggus_parser)
    + [ggus_alerting](#ggus_alerting)
    + [ggus_alert.sh](#ggus_alert.sh)
    + [ggus_alert_manage](#ggus_alert_manage)
  * [Karma Dashboard](#karma-dashboard)
  * [Slack](#slack)
  * [Alert CLI Tool](#alert-cli-tool)
  * [Intelligence Module](#intelligence-module)
    + [The Logic behind intelligence module](#the-logic-behind-intelligence-module)

For detailed information please visit the blog about the whole project.
https://indrarahul.codes/2020/07/24/google-summer-of-code.html

# AlertManagement

## Overview

The growth of distributed services introduces a challenge to properly monitor their status and reduce operational costs. In CMS experiment at CERN we deal with distributed computing infrastructure which includes central services for authentication, workload management, data management, databases, etc. To properly operate and maintain this infrastructure we rely on various open-source monitoring tools, including ElasticSearch, Kafka, Grafana stack (used by central CERN MONIT infrastructure), Prometheus, AlertManager, VictoriaMetrics software components (used by the experiment), as well as custom solutions like GGUS WLCG or ServiceNow ticketing systems.

On daily basis these CMS computing infrastructure may produce significant amount of information about various anomalies, intermittent problems, outages as well as undergo scheduled maintenance. Therefore the amount of alert notifications and tickets our operational teams should handle is very large. We’re working towards, an Operational Intelligent System aiming to detect, analyse, and predict anomalies of the computing environment, to suggest possible actions, and ultimately automate operation procedures. An important component of this system should be an intelligent Alert Management System.

The system should collect anomalies, notifications, etc., from ES, InfluxDB, Prometheus data-sources, GGUS system and be able to aggregate, and create meaningful alerts to address various computing infrastructure failures. For instance, if certain component of global distributed system experiences a problem in terms of its storage the Alert Management system should notify via appropriate channels a set of instructions on how to fix and handle this situation. Before sending the alert it should check if this is a real anomaly or part of on-going outage, or schedule maintenance.

## Architectural Diagram
![Alt text](arch_diag.jpg)

# Installation

You can find the all details to setup this project [here](https://github.com/dmwm/CMSMonitoring/tree/master/doc/AlertManagement/installation.md).

## SSB Alerting Service
Files included in this service are :-
1) [monit](https://github.com/dmwm/CMSMonitoring/blob/master/src/go/MONIT/monit.go)
2) [ssb_alerting](https://github.com/dmwm/CMSMonitoring/blob/master/src/go/MONIT/ssb_alerting.go)
3) [ssb_alert.sh](https://github.com/dmwm/CMSMonitoring/blob/master/scripts/ssb_alert.sh)
4) [ssb_alert_manage](https://github.com/dmwm/CMSMonitoring/blob/master/scripts/ssb_alert_manage)

### monit
The program which is responsible for the SSB queries from InfluxDB. A query can be crafted according to our needs. 

An example of such a query :-

```monit -query="select * from outages where time > now() - 2h" -dbname=monit_production_ssb_otgs -token=$token -dbid=9474```

where we are fetching outages record of last 2 hrs.

### ssb_alerting
monit queries from InfluxDB and dumps the result on stdout which can be dumped on the disk. ssb_alerting fetches dumped SSB JSON Data and converts it to a JSON format which AlertManager's API understands. There could be one issue where some of SSB alerts have no EndTime, i.e. they have open ending. So to solve this issue on each iteration we fetch alerts from AlertManager and check if some alerts having open ending are resolved or not from the maintained HashMap which has information about current alerts. If they are resolved they are pushed with current time as EndTime i.e. we delete those alerts from AlertManager.

```
Usage of ssb_alerting:
  -input string
    	input filename
  -url string
    	alertmanager URL
  -verbose int
    	verbosity level
```
The dataflow and logic behind ssb_alerting tool can be visualized in the diagram below.
![Alt text](alerting.jpg)

### ssb_alert.sh
A simple bash script which makes the above process automated on configurable time interval value.
```
Script for fetching CERN SSB info and injecting it into MONIT (AlertManager)
Usage: ssb_alerts.sh <query> <token> [cmsmon_url] [interval] [verbose]"

  <query>       CMS Monit ES/InfluxDB Query
  <token>       User's Token
   
Options:
  cmsmon_url    CMS Monitoring URL                              (default: https://cms-monitoring.cern.ch)
  interval      Time interval for Alerts ingestion & injection  (default: 1)
  verbose       Verbosity level                                 (default: 0)
  ```
### ssb_alert_manage
A Linux Daemon for our ssb_alerting mechanism which runs our alerting service in the background. It has four open commands like start, stop, status, help to control the ssb alerting service without much of hassle.
Few Environment Variables are required to be set ( PS. some of them have default values.) which makes the whole service easily configurable. 

```
Environments:
  QUERY       :   CMS Monit ES/InfluxDB Query              
  TOKEN       :   User's Token
  CMSMON_URL  :   CMS Monitoring URL                              default - https://cms-monitoring.cern.ch"
  INTERVAL    :   Time interval for Alerts ingestion & injection  default - 1
  VERBOSE     :   Verbosity level                                 default - 0
```
## GGUS Alerting Service
Files included in this service are :-
1) [ggus_parser](https://github.com/dmwm/CMSMonitoring/blob/master/src/go/MONIT/ggus_parser.go)
2) [ggus_alerting](https://github.com/dmwm/CMSMonitoring/blob/master/src/go/MONIT/ggus_alerting.go)
3) [ggus_alert.sh](https://github.com/dmwm/CMSMonitoring/blob/master/scripts/ggus_alert.sh)
4) [ggus_alert_manage](https://github.com/dmwm/CMSMonitoring/blob/master/scripts/ggus_alert_manage)
### ggus_parser
The program which uses HTTP request to fetch GGUS Tickets using CERN Grid Certificate in XML/CSV Format which is then parsed and converted into usable JSON Format and dumped to the disk.

### ggus_alerting
ggus_alerting fetches dumped SSB JSON Data and converts it to a JSON format which AlertManager's API understands. There could be one issue where some of GGUS alerts have no EndTime, i.e. they have open ending. So to solve this issue on each iteration we fetch alerts from AlertManager and check if some alerts having open ending are resolved or not from the maintained HashMap which has information about current alerts. If they are resolved they are pushed with current time as EndTime i.e. we delete those alerts from AlertManager.

```
Usage of ggus_alerting:
  -input string
    	input filename
  -url string
    	alertmanager URL
  -verbose int
    	verbosity level
  -vo string
    	Required VO attribute in GGUS Ticket (default "cms")
```
The dataflow and logic behind ggus_alerting tool can be well visualized in the below diagram. 
![Alt text](alerting.jpg)

### ggus_alert.sh
A simple bash script which makes the above process automated on configurable time interval value.
```
Script for fetching GGUS Tickets and injecting them into MONIT (AlertManager)
Usage: ggus_alerts.sh [ggus_format] [cmsmon_url] [interval] [vo] [timeout] [verbose]"

Options:
  ggus_format   GGUS Query Format ("csv" or "xml")              (default: "csv")
  cmsmon_url    CMS Monitoring URL                              (default: https://cms-monitoring.cern.ch)
  interval      Time interval for Alerts ingestion & injection  (default: 1)
  vo            Required VO attribute                           (default: "cms")
  timeout       HTTP client timeout operation (GGUS Parser)     (default:0 - zero means no timeout)
  verbose       Verbosity level                                 (default: 0)
  ```
### ggus_alert_manage
A Linux Daemon for our ssb_alerting mechanism which runs our alerting service in the background. It has four open commands like start, stop, status, help to control the ssb alerting service without much of hassle.
Few Environment Variables are required to be set ( PS. some of them have default values.) which makes the whole service easily configurable. 

```
Environments:
  GGUS_FORMAT :   GGUS Query Format("csv" or "xml")               default - "csv"
  CMSMON_URL  :   CMS Monitoring URL                              default - https://cms-monitoring.cern.ch"
  INTERVAL    :   Time interval for Alerts ingestion & injection  default - 1
  VO          :   Required VO attribute                           default - "cms"
  TIMEOUT     :   HTTP client timeout operation (GGUS Parser)     default - 0 (zero means no timeout)
  VERBOSE     :   Verbosity level                                 default - 0
```

## Karma Dashboard
"Alertmanager UI is useful for browsing alerts and managing silences, but it’s lacking as a dashboard tool - karma aims to fill this gap."     
-Karma Developer

You also get the URL for alerts which land you to the original ticketing platform. It gives a nice and intuitive view. Multi grid option, collapsing alerts, viewing silences are few nice features of Karma. Below screenshot shows the karma dashboard with alerts from both of the services developed. 
![Alt text](karma.png)

## Slack
Slack has defined channels for particular service alerts. Users are notified about fired alerts which are drived by AlertManager bots. 

![Alt text](slack.png)

## Alert CLI Tool

Nice and clean CLI interface for getting alerts, their details on terminal either in tabular form or JSON format. Convenient option for operators who prefer command line tools. Comes with several options such as :-
 - service, severity, tag - Filters
 - sort - Sorting
 - name - For detailed information of an alert
 - json - information in JSON format

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

Below is a screenshot of such queries using the tool.
![Alt text](alert.png)

## Intelligence Module

It is a data pipeline. Each components are independent of each other. One component receives the data, adds its logic and forwards the processed data to other component.

Below is the Intelligence module architecture diagram.
![Alt text](int_mod.png)

What it does ?
 - assigns proper severity levels to SSB/GGUS alerts which helps operators to understand the criticality of the infrastructure. Ex. If Number of Alerts with severity=”urgent” > some threshold, then the infrastructure is in critical situation.
- annotates Grafana Dashboards when Network or Database interventions.
- predicts type of alerts and groups similar alerts with the help of Machine Learning.
- adds applicable tutorial/instructions URL/doc to alert, on following which an operator can solve the issue quickly.

### The Logic behind intelligence module
1) The main function will run all pipeline's logic.
2) pipeline starts with fetching alerts,
3) then to process each alert seeing if it has already been processed and silenced. If they are then we ignore that alert, otherwise, we pass it to the next pipeline component. ( So here we require the SilencedMap that stores all those alerts which are in Silence Mode, indicates they are processed and we should not repeat the intelligence process again for them ).
4) Then the alert comes to keyword matching were keywords are matched and accordingly severity level is assigned.
5) Passes through ML Box with no logic as of now.
6) then the processed alert is pushed to AM and,
7) the old alert with default severity is silenced
8) Then Some resolved alerts that are silenced (i.e. when GGUS alerts are resolved) are deleted. This step for those alerts which have open ending. We wait for them till they are resolved.

Testing:-

We use counters to verify if the intelligence module testing was successful or not. If there is mismatch in the counters we declare failure. We also check if the module successfully annotated the dashboards or not. To pass the test, the module shouldn't make random values in counters and it should annotate the dashboards.

- When we are fetching the alerts at step 2, we will count the number of alerts in the AM (i.e. before intelligence module does its stuff)
- Then when we go to preprocessing step 3, we will count all active, expired silences when we update our SilenceMap.
- When we create new Silences we keep adding 1 to the Active Silence counter which we modified at step 3.
- When we delete a resolved alert's Silence we keep adding 1 to the Expired Silence counter which we modified at step 3.

Now at the end of pipeline. We will end up having following counters:-

- No Of Active Silences
- No Of Expired Silences
- No of Pending Silences

Now to verify if everything went well at the end of pipeline. We will again fetch Alerts/Silences from AM and count them and check if they matches with the counters above. If they match, check for the "AnnotateTestStatus" value, if the module has annnotated the dashboards, it will be true and thus Test will be succesfull or if any one of them fails, Testing fails.

**MANUAL CHECK**
Also look on number of alerts counter. Although we can't say before and after running of intelligence module this counter should be equal because any time new alert can come in and we may be in the middle of run of our intelligence module. But we do know the number of alerts after running the intelligence module should not go high (exponential) as limited number of alerts are created at GGUS/SSB. So if you see unexpectedly high increase in number of alerts after running the intelligence module, consider the testing failed. 

Detailed instructions how to setup, run and test the intelligence module can be found [here](https://github.com/dmwm/CMSMonitoring/blob/master/src/go/intelligence/README.md).

We also are providing solution for silencing false alerts which are created due to maintenance alert. The maintenance alert has a field of instances which is a list of all those instance which are going to get affected due to this ongoing maintenance. So our [intelligence program](https://github.com/dmwm/CMSMonitoring/blob/master/src/go/MONIT/intelligence.go), goes to Alertmanager fetches all alerts and filters them based on the searchingLabel (here instance) and silence them for time the maintenance is going. Once the maintenance ends the if alerts are still alive they come back to the AlertManager. This helps the operator to deal with less alerts when maintenance is going on, he/she will look into the maintenance alerts only, not all those false alerts fired due to maintenance.

For installation walkthrough of this program go [here](https://github.com/dmwm/CMSMonitoring/tree/master/doc/AlertManagement/installation.md#intelligence-program).
