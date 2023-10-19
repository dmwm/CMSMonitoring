import os
import json
import time
import logging
from .StompAMQ7 import StompAMQ7


def credentials(f_name):
    if os.path.exists(f_name):
        return json.load(open(f_name))
    return {}


def drop_nulls_in_dict(d):  # d: dict
    """Drops the dict key if the value is None

    ES mapping does not allow None values and drops the document completely.
    """
    return {k: v for k, v in d.items() if v is not None}  # dict


def to_chunks(data, samples=1000):
    length = len(data)
    for i in range(0, length, samples):
        yield data[i : i + samples]


def send_to_amq(data, confs, batch_size, topic=None, doc_type=None, overwrite_meta_ts=False):
    """Sends list of dictionary in chunks"""
    wait_seconds = 0.001
    if confs:
        username = confs.get("username", "")
        password = confs.get("password", "")
        producer = confs.get("producer")
        if not doc_type:
            doc_type = confs.get("type")
        if not topic:
            topic = confs.get("topic")
        host = confs.get("host")
        port = int(confs.get("port"))
        cert = confs.get("cert")
        ckey = confs.get("ckey")
        for chunk in to_chunks(data, batch_size):
            stomp_amq = StompAMQ7(
                username=username,
                password=password,
                producer=producer,
                topic=topic,
                key=ckey,
                cert=cert,
                validation_schema=None,
                host_and_ports=[(host, port)],
                loglevel=logging.WARNING,
            )
            messages = []
            for msg in chunk:
                ts = None if not overwrite_meta_ts else msg.get("timestamp")
                notif, _, _ = stomp_amq.make_notification(
                    payload=msg, doc_type=doc_type, producer=producer, ts=ts
                )
                messages.append(notif)
            if messages:
                stomp_amq.send(messages)
                time.sleep(wait_seconds)
        time.sleep(1)
        print("Message sending is finished")
