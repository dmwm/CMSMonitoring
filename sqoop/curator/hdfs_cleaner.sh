#!/bin/bash
# Retention policy cleaner for CMS Monitoring HDFS paths
# shellcheck disable=SC2181

export PATH=$PATH:/usr/hdp/hadoop/bin

#######################################
# Deletes HDFS directory and print meaningful message
# Arguments:
#   arg1: HDFS directory
#   arg2: output message
#######################################
function util_hdfs_delete() {
    local dir msg
    dir=$1
    msg=$2
    hdfs dfs -test -e "$dir" >/dev/null 2>&1
    if [ $? -eq 0 ]; then
        hdfs dfs -rm -r -skipTrash "$dir" >/dev/null 2>&1
        if [ $? -eq 0 ]; then
            echo "<HDFS retention policy deletion> $dir deleted. $msg"
        else
            echo "<HDFS retention policy deletion> Unsuccessful. $msg"
        fi
    fi
}

kinit -R

# RUCIO (rucio_table_dumps.sh | /project/awg/cms/rucio) 18x30 days retention policy
rucio_month_retention="540 days ago"
rucio_base_dir="/project/awg/cms/rucio"
rucio_month=$(date -d "$rucio_month_retention" +%Y-%m-%d)
util_hdfs_delete "$rucio_base_dir/$rucio_month" "RUCIO"

# DBS (dbs3_full_global.sh | /project/awg/cms/dbs/PROD_GLOBAL) 6x30 days retention policy
dbs3_month_retention="180 days ago"
dbs3_dir="/project/awg/cms/dbs/PROD_GLOBAL"
dbs3_month=$(date -d "$dbs3_month_retention" +%Y-%m-%d)
util_hdfs_delete "$dbs3_dir/$dbs3_month" "DBS3 PROD_GLOBAL"

# DBS FILE_LUMIS (dbs3_full_global.sh | /project/awg/cms/dbs/PROD_GLOBAL/*/FILE_LUMIS) 30 days retention policy
dbs_file_lumis_day_retention="30 days ago"
dbs_file_lumis_dir="/project/awg/cms/dbs/PROD_GLOBAL"
dbs_file_lumis_day=$(date -d "$dbs_file_lumis_day_retention" +%Y-%m-%d)
util_hdfs_delete "$dbs_file_lumis_dir/$dbs_file_lumis_day/FILE_LUMIS" "DBS3 FILE_LUMIS"

# TASKS (cms-tasks.sh | /project/awg/cms/crab/tasks) 30 days retention policy
tasks_day_retention="30 days ago"
tasks_dir="/project/awg/cms/crab/tasks"
tasks_day=$(date -d "$tasks_day_retention" +%Y-%m-%d)
util_hdfs_delete "$tasks_dir/$tasks_day" "TASKS"
