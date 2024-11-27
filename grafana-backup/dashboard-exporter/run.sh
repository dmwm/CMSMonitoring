#!/bin/bash -l

set -e
# shellcheck disable=SC1090

# This script copies Grafana dashboard jsons, tars them and puts into EOS folder.
# Ref for alerting: https://github.com/dmwm/CMSSpark/blob/master/bin/cron4aggregation

##H Usage: run.sh KERBEROS_KEYTAB TOKEN_LOCATION FILESYSTEM_PATH
##H Example:
##H        run.sh keytab keys/token.json /eos/cms/store/group/offcomp_monit/

# help definition
if [ "$1" == "-h" ] || [ "$1" == "-help" ] || [ "$1" == "--help" ] || [ "$1" == "help" ] || [ "$1" == "" ]; then
    grep "^##H" <"$0" | sed -e "s,##H,,g"
    exit 1
fi

# Get utils functions
. ../../scripts/utils.sh

# Authenticate with Kerberos keytab
util_kerberos_auth_with_keytab "$1"

# Change working directory
cd "$(dirname "$0")" || exit

addr=cms-comp-monit-alerts@cern.ch

# Call func function on exit
trap onExit exit

# Define func
function onExit() {
  local status=$?
  if [ $status -ne 0 ]; then
    local msg="Grafana backup cron failure. Please see Kubernetes 'cron' cluster 'hdfs' namespace logs."
    if [ -f ./amtool ]; then
      expire=$(date -d '+1 hour' --rfc-3339=ns | tr ' ' 'T')
      local expire
      local urls="http://cms-monitoring.cern.ch:30093 http://cms-monitoring-ha1.cern.ch:30093 http://cms-monitoring-ha2.cern.ch:30093"
      for url in $urls; do
        ./amtool alert add grafana_backup_failure \
          alertname=grafana_backup_failure severity=monitoring tag=cronjob alert=amtool \
          --end="$expire" \
          --annotation=summary="$msg" \
          --annotation=date="$(date)" \
          --annotation=hostname="$(hostname)" \
          --annotation=status="$status" \
          --alertmanager.url="$url"
      done
    else
      echo "$msg" | mail -s "Cron alert grafana_backup_failure" "$addr"
    fi
  fi
}

# Execute
./dashboard-exporter.py --token "$2" --filesystem-path "$3"
