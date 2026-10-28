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


def _is_unevaluated_classad(value):
    if value is None:
        return False
    try:
        import classad  # pylint: disable=import-outside-toplevel

        if isinstance(value, (classad.ExprTree, classad.ClassAd, classad.Value)):
            return True
    except ImportError:
        pass

    type_name = type(value).__name__
    module = getattr(type(value), "__module__", "") or ""
    return type_name in ("ExprTree", "ClassAd", "Value") and "classad" in module


def _safe_expression_text(value):
    try:
        return str(value)
    except Exception as exc:  # pylint: disable=broad-except
        return "<unable to stringify expression: %s>" % exc


def _sanitize_payload(payload, field_path="", stripped=None):
    if stripped is None:
        stripped = []

    if _is_unevaluated_classad(payload):
        stripped.append((field_path or "<root>", _safe_expression_text(payload)))
        return None

    if isinstance(payload, dict):
        return {
            key: _sanitize_payload(
                value,
                field_path=("%s.%s" % (field_path, key)) if field_path else key,
                stripped=stripped,
            )
            for key, value in payload.items()
        }
    if isinstance(payload, (list, tuple)):
        return [
            _sanitize_payload(
                value,
                field_path="%s[%d]" % (field_path, index),
                stripped=stripped,
            )
            for index, value in enumerate(payload)
        ]

    return payload


def prepare_job_payload(payload):
    """Return payload safe for upload, with a list of stripped ClassAd fields."""
    stripped = []
    clean = _sanitize_payload(payload, stripped=stripped)
    return clean, stripped


def _log_amq_notification_failure(doc_id, ad, exc, stripped=None):
    global_job_id = ad.get("GlobalJobId", doc_id)
    if stripped is None:
        stripped = []
    expression_details = "; ".join(
        "%s=%r" % (path, expression) for path, expression in stripped
    )
    if expression_details:
        logging.error(
            "AMQ notification failed for job %s (doc_id=%s); "
            "unevaluated ClassAd expressions: %s; error: %s",
            global_job_id,
            doc_id,
            expression_details,
            exc,
            exc_info=True,
        )
    else:
        logging.error(
            "AMQ notification failed for job %s (doc_id=%s): %s",
            global_job_id,
            doc_id,
            exc,
            exc_info=True,
        )


def post_ads(ads, metadata=None):
    if not len(ads):
        logging.warning("No new documents found")
        return 0, 0, 0.0

    metadata = metadata or {}
    interface = get_amq_interface()
    list_data = []
    for id_, ad in ads:
        try:
            notif, _, _ = interface.make_notification(
                payload=ad,
                doc_type=None,  # will be default "metric"
                doc_id=id_,
                ts=recordTime(ad),
                metadata=metadata,
                data_subfield=None,
            )
            list_data.append(notif)
        except Exception as exc:
            _, stripped = prepare_job_payload(ad)
            _log_amq_notification_failure(id_, ad, exc, stripped=stripped)
            raise

    starttime = time.time()
    failed_to_send = interface.send(list_data)
    elapsed = time.time() - starttime
    if failed_to_send:
        raise RuntimeError(
            "Failed to send %d/%d notifications to AMQ"
            % (len(failed_to_send), len(list_data))
        )
    return len(ads), len(ads), elapsed
