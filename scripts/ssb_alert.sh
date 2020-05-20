#!/bin/bash
##H Script for fetching CERN SSB info and injecting it into MONIT (AlertManager)
##H Usage: ssb_alerts.sh [-u url] [-i interval] [-v level] <ssb_query> <token-file>"
##H
##H Options:
##H   -u url        alertmanager url
##H   -i interval   time interval (in sec) at which fetching and injecting data repeats
##H   -v level      verbosity level    
##H

# Check if user is passing least required arguments.
if [ "$#" -lt 2  ]; then
    perl -ne '/^##H/ && do { s/^##H ?//; print }' < $0
    exit 1
fi

query="$1"
token=$2

# Alerting Tool optional arguments
alertmanager_url=${3:-"https://cms-monitoring.cern.ch"}
interval=${4:-1}
verbose=${5:-0}

input_file=data.json

while true;  do
   monit -query="$query" -dbname=monit_production_ssb_otgs -token=$token -dbid=9474 > $input_file
   alerting -input $input_file -url $alertmanager_url -verbose $verbose
   sleep $interval
done
