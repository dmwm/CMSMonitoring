#!/bin/sh
##H Script for fetching CERN SSB info and injecting it into MONIT (AlertManager)
##H Usage: ssb_alerts.sh <query> <token> [data_file] [cmsmon_url] [interval] [verbose]"
##H
##H   <query>       CMS Monit ES/InfluxDB Query
##H   <token>       User's Token
##H    
##H Options:
##H   data_file     data json file to use                           (default: /tmp/ssb_data.json)
##H   cmsmon_url    CMS Monitoring URL                              (default: https://cms-monitoring.cern.ch)
##H   interval      Time interval for Alerts ingestion & injection  (default: 1)
##H   verbose       Verbosity level                                 (default: 0)
##H

# Check if user is passing least required arguments.
if [ "$#" -lt 2  ]; then
    cat $0 | grep "^##H" | sed -e "s,##H,,g"
    exit 1
fi

query="$1"
token=$2

# Alerting Tool optional arguments
data_file=${3:-"/tmp/ssb_data.json"}
cmsmon_url=${4:-"https://cms-monitoring.cern.ch"}
interval=${5:-1}
verbose=${6:-0}

while true;  do
    echo "### monit -query="$query" -dbname=monit_production_ssb_otgs -token=$token -dbid=9474"
    monit -query="$query" -dbname=monit_production_ssb_otgs -token=$token -dbid=9474 > $data_file
    echo "### ssb_alerting -input $data_file -url $cmsmon_url -verbose $verbose"
    ssb_alerting -input $data_file -url $cmsmon_url -verbose $verbose
    sleep $interval
done
