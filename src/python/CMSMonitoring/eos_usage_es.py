# !/usr/bin/env python
# -*- coding: utf-8 -*-
# Author: Ceyhun Uzunoglu <ceyhunuzngl AT gmail [DOT] com>
# Sends CMS EOS summary calculations to MONIT in each 10 minute
#
# Cron script CMSMOnitoring/scripts/cron4eos_usage_es.sh
import json
import logging
import os
import sys
import time
import click

# CMSMonitoring modules
try:
    from CMSMonitoring.StompAMQ7 import StompAMQ7
except ImportError:
    print("ERROR: Could not import StompAMQ")
    sys.exit(1)


def credentials(f_name):
    if os.path.exists(f_name):
        return json.load(open(f_name))
    return {}


def to_chunks(data, samples=1000):
    length = len(data)
    for i in range(0, length, samples):
        yield data[i:i + samples]


def special_send_to_amq(data, confs, batch_size):
    """Sends list of dictionary in chunks"""
    ts = int(time.time()) * 1000
    wait_seconds = 0.001
    if confs:
        username = confs.get('username', '')
        password = confs.get('password', '')
        producer = confs.get('producer')
        topic = confs.get('topic')
        doc_type = confs.get('type', None)
        host = confs.get('host')
        port = int(confs.get('port'))
        cert = confs.get('cert', None)
        ckey = confs.get('ckey', None)
        for chunk in to_chunks(data, batch_size):
            # After each stomp_amq.send, we need to reconnect with this way.
            stomp_amq = StompAMQ7(username=username, password=password, producer=producer, topic=topic,
                                  key=ckey, cert=cert, validation_schema=None, host_and_ports=[(host, port)],
                                  loglevel=logging.WARNING)
            messages = []
            for msg in chunk:
                # Set metadata.timestamp as tstamp_hour of the old data
                notif, _, _ = stomp_amq.make_notification(payload=msg, doc_type=doc_type,
                                                          producer=producer, ts=ts)
                messages.append(notif)
            if messages:
                stomp_amq.send(messages)
                time.sleep(wait_seconds)
        time.sleep(1)
        print("Message sending is finished")


def get_data(json_file):
    with open(json_file) as f:
        return json.loads(f.read())


@click.command()
@click.option("--creds", required=True, help="secret file path: secrets/cms-eos-mon/amq_broker.json")
@click.option("--summary_json", required=True, help="/eos/cms/store/eos_accounting_summary.json")
def main(creds, summary_json):
    """Main function that sends data to MONIT
    """
    creds_json = credentials(f_name=creds)
    special_send_to_amq(data=get_data(summary_json), confs=creds_json, batch_size=10000)


if __name__ == "__main__":
    main()
