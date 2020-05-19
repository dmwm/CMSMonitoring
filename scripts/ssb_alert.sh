#!/bin/bash
##H Script for fetching CERN SSB info and injecting it into MONIT (AlertManager)
##H Usage: ssb_alerts.sh [-u url] [-i interval] [-v level] <ssb_query> <token-file>"
##H
##H Options:
##H   -u url        alertmanager url
##H   -i interval   time interval (sec)
##H   -v level      verbosity level    
##H


# Alerting Tool default arguments
alertmanager_url="http://localhost:9093"
interval=1
verbose=0

# Alerting Tool optional arguments parsing logic
while getopts ":u:i:v:" opt; do
  case ${opt} in
    u )
      alertmanager_url=$OPTARG
    ;;
    i)
      interval=$OPTARG
    ;;
    v)
      verbose=$OPTARG
    ;;
    \? )
      echo "Invalid Option: -$OPTARG" 1>&2
      perl -ne '/^##H/ && do { s/^##H ?//; print }' < $0
      exit 1
      ;;
  esac
done
shift $((OPTIND -1))

# Check if user is passing least required arguments.
if [ "$#" -lt 2  ]; then
    perl -ne '/^##H/ && do { s/^##H ?//; print }' < $0
    exit 1
fi

query="$1"
token=$2

# Alerting Tool arguments
input_file=data.json

while true;  do
   monit -query="$query" -dbname=monit_production_ssb_otgs -token=$token -dbid=9474 > $input_file
   alerting -input $input_file -url $alertmanager_url -verbose $verbose
   sleep $interval
done