#!/bin/bash

#Check if user is passing all arguments.
if [ "$#" != 5  ]; then
    echo "ssb_alerts.sh provides basic daemon to fetch CERN SSB info and inject it into MONIT (AlertManager) "
    echo "Usage: ssb_alerts.sh <ssb_query> <token-file> <alertmanager-url> <interval (in sec)> <verbosity level>"
    exit 1
fi

query=$1
token=$2
alertmanager_url=$3
interval=$4
verbose=$5

# Alerting Tool arguments
input_file=data.json

while true;  do
   monit -query="$query" -dbname=monit_production_ssb_otgs -token=$token -dbid=9474 > $input_file
   alerting -input $input_file -url $alertmanager_url -verbose $verbose
   sleep $interval
done