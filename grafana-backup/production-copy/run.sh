#!/bin/bash -l

set -e

##H Usage: run.sh GRAFANA_URL API_TOKEN
##H Example:
##H        run.sh https://grafana-url.com your-api-token

if [ "$1" == "-h" ] || [ "$1" == "-help" ] || [ "$1" == "--help" ] || [ "$1" == "help" ] || [ "$1" == "" ]; then
    grep "^##H" <"$0" | sed -e "s,##H,,g"
    exit 1
fi

# Arguments
GRAFANA_URL="$1"
API_TOKEN="$2"
shift 2

addr=cms-comp-monit-alerts@cern.ch

trap onExit EXIT

function onExit() {
  local status=$?
  if [ $status -ne 0 ]; then
    local msg="Grafana dashboard sync job failure. Please see Kubernetes 'cron' cluster logs."
    if [ -x ./amtool ]; then
      expire=$(date -d '+1 hour' --rfc-3339=ns | tr ' ' 'T')
      local urls="http://cms-monitoring.cern.ch:30093 http://cms-monitoring-ha1.cern.ch:30093 http://cms-monitoring-ha2.cern.ch:30093"
      for url in $urls; do
        ./amtool alert add grafana_dashboard_sync_failure \
          alertname=grafana_dashboard_sync_failure severity=critical tag=cronjob alert=amtool \
          --end="$expire" \
          --annotation=summary="$msg" \
          --annotation=date="$(date)" \
          --annotation=hostname="$(hostname)" \
          --annotation=status="$status" \
          --alertmanager.url="$url"
      done
    else
      echo "$msg" | mail -s "Cron alert grafana_dashboard_sync_failure" "$addr"
    fi
  fi
}

cd "$(dirname "$0")" || exit

# Execute the Python script
python3 dashboard-copy.py --url "$GRAFANA_URL" --token "$API_TOKEN"
