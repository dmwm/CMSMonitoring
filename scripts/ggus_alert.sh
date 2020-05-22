#!/bin/bash
##H Script for fetching GGUS Tickets and injecting them into MONIT (AlertManager)
##H Usage: ggus_alerts.sh [ggus_format] [cmsmon_url] [interval] [vo] [timeout] [verbose]"
##H
##H Options:
##H   ggus_format   GGUS Query Format ("csv" or "xml")              (default: "csv")
##H   cmsmon_url    CMS Monitoring URL                              (default: https://cms-monitoring.cern.ch)
##H   interval      Time interval for Alerts ingestion & injection  (default: 1)
##H   vo            Required VO attribute                           (default: "cms")
##H   timeout       HTTP client timeout operation (GGUS Parser)     (default:0 - zero means no timeout)
##H   verbose       Verbosity level                                 (default: 0)
##H

# Check if user is passing least required arguments.
if [ "$#" -lt 1  ]; then
    perl -ne '/^##H/ && do { s/^##H ?//; print }' < $0
    exit 1
fi


ggus_format=${1:-"csv"}

## Alerting Tool optional arguments
cmsmon_url=${2:-"https://cms-monitoring.cern.ch"}
interval=${3:-1}
vo=${4:-"cms"}
##

## GGUS Parser optional arguments
timeout=${5:-0}
##

verbose=${6:-0}

data_file=data.json

while true;  do
   ggus_parser -format $ggus_format -out $data_file -timeout $timeout -verbose $verbose
   ggus_alerting -input $data_file -url $cmsmon_url -vo $vo -verbose $verbose
   sleep $interval
done
