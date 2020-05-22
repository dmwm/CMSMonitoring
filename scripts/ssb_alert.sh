#!/bin/bash
##H Script for fetching CERN SSB info and injecting it into MONIT (AlertManager)
##H Usage: ssb_alerts.sh <query> <token> [cmsmon_url] [interval] [verbose]"
##H
##H   <query>       CMS Monit ES/InfluxDB Query
##H   <token>       User's Token
##H    
##H Options:
##H   cmsmon_url    CMS Monitoring URL                              (default: https://cms-monitoring.cern.ch)
##H   interval      Time interval for Alerts ingestion & injection  (default: 1)
##H   verbose       Verbosity level                                 (default: 0)
##H

# Check if user is passing least required arguments.
if [ "$#" -lt 2  ]; then
    perl -ne '/^##H/ && do { s/^##H ?//; print }' < $0
    exit 1
fi

query="$1"
token=$2

# Alerting Tool optional arguments
cmsmon_url=${3:-"https://cms-monitoring.cern.ch"}
interval=${4:-1}
verbose=${5:-0}

data_file=data.json

while true;  do
   monit -query="$query" -dbname=monit_production_ssb_otgs -token=$token -dbid=9474 > $data_file
   ssb_alerting -input $data_file -url $cmsmon_url -verbose $verbose
   sleep $interval
done
