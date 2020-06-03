#!/bin/sh
##H Script for fetching GGUS Tickets and injecting them into MONIT (AlertManager)
##H Usage: ggus_alerts.sh [ggus_format] [data_file] [cmsmon_url] [interval] [vo] [timeout] [verbose]"
##H
##H Options:
##H   ggus_format   GGUS Query Format ("csv" or "xml")              (default: "csv")
##H   data_file     data json file to use                           (default: "/tmp/data.json")
##H   cmsmon_url    CMS Monitoring URL                              (default: https://cms-monitoring.cern.ch)
##H   interval      Time interval for Alerts ingestion & injection  (default: 1)
##H   vo            Required VO attribute                           (default: "cms")
##H   timeout       HTTP client timeout operation (GGUS Parser)     (default:0 - zero means no timeout)
##H   verbose       Verbosity level                                 (default: 0)
##H

# Check if user is passing least required arguments.
if [ "$#" -lt 1  ]; then
    cat $0 | grep "^##H" | sed -e "s,##H,,g"
    exit 1
fi


ggus_format=${1:-"csv"}
data_file=${2:-"/tmp/data.json"}

## Alerting Tool optional arguments
cmsmon_url=${3:-"https://cms-monitoring.cern.ch"}
interval=${4:-1}
vo=${5:-"cms"}
##

## GGUS Parser optional arguments
timeout=${6:-0}
##

verbose=${7:-0}

while true;  do
   ggus_parser -format $ggus_format -out $data_file -timeout $timeout -verbose $verbose
   ggus_alerting -input $data_file -url $cmsmon_url -vo $vo -verbose $verbose
   sleep $interval
done
