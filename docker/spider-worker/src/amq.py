import time
import logging
from CMSMonitoring.StompAMQ7 import StompAMQ7 as StompAMQ

import constants as const
# TODO: remove this dependency
from convert_to_json import recordTime

_amq_interface = None


def get_amq_interface():
    global _amq_interface
    if not _amq_interface:
        _amq_interface = StompAMQ(
            username=const.AMQ_USERNAME,
            password=const.AMQ_PASSWORD,
            producer=const.AMQ_PRODUCER,
            topic=const.AMQ_TOPIC,
            host_and_ports=[(const.AMQ_BROKER, const.AMQ_PORT)],
            validation_schema=const.SPIDER_VALIDATION_SCHEMA,
        )

    return _amq_interface


def post_ads(ads, metadata=None):
    if not len(ads):
        logging.warning("No new documents found")
        return

    metadata = metadata or {}
    interface = get_amq_interface()
    list_data = []
    for id_, ad in ads:
        notif, _, _ = interface.make_notification(
            payload=ad,
            doc_type=None,  # will be default "metric"
            doc_id=id_,
            ts=recordTime(ad),
            metadata=metadata,
            data_subfield=None,
        )
        list_data.append(notif)

    starttime = time.time()
    failed_to_send = interface.send(list_data)
    elapsed = time.time() - starttime
    return len(ads) - len(failed_to_send), len(ads), elapsed
