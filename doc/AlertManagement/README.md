# Content

- [AlertManagement](#alertmanagement)
  * [Overview](#overview)
  * [Architectural Diagram](#architectural-diagram)
  * [SSB Alerting Service](#ssb-alerting-service)
    + [monit](#monit)
    + [ssb_alerting](#ssb-alerting)
    + [ssb_alert.sh](#ssb-alertsh)
    + [ssb_alert_manage](#ssb-alert-manage)
  * [GGUS Alerting Service](#ggus-alerting-service)
    + [ggus_parser](#ggus-parser)
    + [ggus_alerting](#ggus-alerting)
    + [ggus_alert.sh](#ggus-alertsh)
    + [ggus_alert_manage](#ggus-alert-manage)
  * [Karma Dashboard](#karma-dashboard)
  * [Intelligence Module](#intelligence-module)
    + [The Logic behind intelligence module](#the-logic-behind-intelligence-module)
    
For more detailed information please visit the blog about the whole project.
https://indrarahul.codes/2020/07/24/google-summer-of-code.html

# AlertManagement

## Overview

The growth of distributed services introduces a challenge to properly monitor their status and reduce operational costs. In CMS experiment at CERN we deal with distributed computing infrastructure which includes central services for authentication, workload management, data management, databases, etc. To properly operate and maintain this infrastructure we rely on various open-source monitoring tools, including ElasticSearch, Kafka, Grafana stack (used by central CERN MONIT infrastructure), Prometheus, AlertManager, VictoriaMetrics software components (used by the experiment), as well as custom solutions like GGUS WLCG or ServiceNow ticketing systems.

On daily basis these CMS computing infrastructure may produce significant amount of information about various anomalies, intermittent problems, outages as well as undergo scheduled maintenance. Therefore the amount of alert notifications and tickets our operational teams should handle is very large. We’re working towards, an Operational Intelligent System aiming to detect, analyse, and predict anomalies of the computing environment, to suggest possible actions, and ultimately automate operation procedures. An important component of this system should be an intelligent Alert Management System.

The system should collect anomalies, notifications, etc., from ES, InfluxDB, Prometheus data-sources, GGUS system and be able to aggregate, and create meaningful alerts to address various computing infrastructure failures. For instance, if certain component of global distributed system experiences a problem in terms of its storage the Alert Management system should notify via appropriate channels a set of instructions on how to fix and handle this situation. Before sending the alert it should check if this is a real anomaly or part of on-going outage, or schedule maintenance.

## Architectural Diagram
![Alt text](arch_diag.jpg)

## SSB Alerting Service
Files included in this service are :-
1) monit
2) ssb_alerting
3) ssb_alert.sh
4) ssb_alert_manage

### monit
The program which is responsible for the SSB queries from InfluxDB. A query can be manually drafted according to our needs. 

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
The dataflow and logic behind ssb_alerting tool can be well visualized in the below diagram.
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
A Linux Daemon for our ssb_alerting mechanism which makes possible to run our service in background giving access to major commands like start, stop, status, help.
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
1) ggus_parser
2) ggus_alerting
3) ggus_alert.sh
4) ggus_alert_manage

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
A Linux Daemon for the ggus_alerting mechanism which makes possible to run our service in background giving access to major commands like start, stop, status, help.
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

A user has to set the required Environment variables. Once all environment Variables are set you can run these two commands only to start the services. That's it.

``` $ ggus_alert_manage start ```
``` $ ssb_alert_manage start ```

## Karma Dashboard
"Alertmanager UI is useful for browsing alerts and managing silences, but it’s lacking as a dashboard tool - karma aims to fill this gap."     
-Karma Developer

We can build the docker container using Karma Dockerfile.

```docker build -t <CERN_REPO./karma <path-to-Dockerfile>```

Required Environment Variable for docker container -

```ALERTMANAGER_URI ```

Karma Dashboard can be configured using `karma.yaml` file.
```
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

Below screenshot shows the karma dashboard with alerts from both of the services developed.
![Alt text](karma.png)

## Intelligence Module

Detailed instructions how to setup, run and test the intelligence module can be found [here](https://github.com/dmwm/CMSMonitoring/blob/master/src/go/intelligence/README.md).

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
