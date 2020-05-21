#!/bin/bash
##H Script for fetching GGUS Tickets and injecting them into MONIT (AlertManager)
##H Usage: ggus_alerts.sh <ggus_format> [url] [interval] [level]"
##H
##H Options:
##H   url        alertmanager url (default: https://cms-monitoring.cern.ch)
##H   interval   time interval (in sec) at which fetching and injecting data repeats (default: 1)
##H   verbose    verbosity level (int) (default: 0)
##H

# Check if user is passing least required arguments.
if [ "$#" -lt 1  ]; then
    perl -ne '/^##H/ && do { s/^##H ?//; print }' < $0
    exit 1
fi


ggus_format=${1:-"csv"}

# Alerting Tool optional arguments
alertmanager_url=${2:-"https://cms-monitoring.cern.ch"}
interval=${3:-1}
vo=${4:-"cms"}
verbose=${5:-0}

input_file=data.json

while true;  do
   ggus_parser -format $ggus_format -out $input_file
   ggus_alerting -input $input_file -url $alertmanager_url -vo $vo -verbose $verbose
   sleep $interval
done
