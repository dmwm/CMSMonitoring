#!/bin/bash
# Copied from vocms0240
#
export SPIDER_WORKDIR="/opt/spider"
export AFFILIATION_DIR_LOCATION="$SPIDER_WORKDIR/.affiliation_dir.json"
export PYTHONPATH="$SPIDER_WORKDIR/src/:$PYTHONPATH"
export CMS_HTCONDOR_TOPIC="/topic/cms.jobmon.condor"

# PROD
export CMS_HTCONDOR_PRODUCER="condor"
export CMS_HTCONDOR_BROKER="cms-mb.cern.ch"
_LOGDIR=$SPIDER_WORKDIR/log/
_LOG_LEVEL="WARNING"

_QUERY_QUEUE_BATCH_SIZE=100
_QUERY_POOL_SIZE=16
_UPLOAD_POOL_SIZE=8

cd $SPIDER_WORKDIR || exit
source "$SPIDER_WORKDIR/venv/bin/activate"

# ./scripts/cronAffiliation.sh # First run

python "$SPIDER_WORKDIR/spider_cms.py" \
    --log_dir $_LOGDIR \
    --log_level $_LOG_LEVEL \
    --skip_history \
    --process_queue \
    --query_queue_batch_size $_QUERY_QUEUE_BATCH_SIZE \
    --query_pool_size $_QUERY_POOL_SIZE \
    --upload_pool_size $_UPLOAD_POOL_SIZE \
    --collectors_file $SPIDER_WORKDIR/etc/collectors.json

#python spider_cms.py --log_dir $LOGDIR --log_level WARNING --feed_amq --email_alerts 'cms-comp-monit-alerts@cern.ch' --skip_history --process_queue --query_queue_batch_size 100 --query_pool_size 16 --upload_pool_size 8 --collectors_file $SPIDER_WORKDIR/etc/collectors.json

# crontab entry (to run every 12 min):
# */12 * * * * /opt/spider/scripts/spider_cms.sh
