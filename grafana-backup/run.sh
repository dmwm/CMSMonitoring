#!/bin/bash -l
# shellcheck disable=SC1090

# This script copies Grafana dashboard jsons, tar them and put into hdfs folder.
# Ref for alerting: https://github.com/dmwm/CMSSpark/blob/master/bin/cron4aggregation

##H Usage: run.sh TOKEN_LOCATION HDFS_PATH FILESYSTEM_PATH
##H Example:
##H        run.sh keys/token.json /cms/backups/grafana/ /eos/cms/store/group/offcomp_monit/

# help definition
if [ "$1" == "-h" ] || [ "$1" == "-help" ] || [ "$1" == "--help" ] || [ "$1" == "help" ] || [ "$1" == "" ]; then
    grep "^##H" <"$0" | sed -e "s,##H,,g"
    exit 1
fi

# Change working directory
cd "$(dirname "$0")" || exit

addr=ceyhun.uzunoglu@cern.ch

apath=/data/cms/anaconda3
export PATH=$PATH:$apath/bin/
if [ -f $apath/etc/profile.d/conda.sh ]; then

  source $apath/etc/profile.d/conda.sh
fi
conda activate py3-stomp

# Add cvmfs envs to run amtool
export VO_CMS_SW_DIR=/cvmfs/cms.cern.ch
source $VO_CMS_SW_DIR/cmsset_default.sh

# Call func function on exit
trap onExit exit

# Define func
function onExit() {
  local status=$?
  if [ $status -ne 0 ]; then
    local msg="Grafana backup cron failure. Please see vocms092:/data/cms/cmsmonitoringbackup/log"
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
/data/cms/cmsmonitoringbackup/dashboard-exporter.py --token $1 --hdfs-path $2 --filesystem-path $3
