#!/usr/bin/env python
# -*- coding: utf-8 -*-
# pylint: disable=C0301,R0903,R0912,W0703,C0103,R0914,W0603,R0902,R0913
"""
Basic interface to CERN ActiveMQ via stomp6+
References:
    - MONIT reference https://monit-docs.web.cern.ch/metrics/amq/#sending-data
"""
import json
import logging
import os
import random
import socket
import stomp
import sys
import time
import traceback
from time import gmtime
from uuid import uuid4

from stomp.exception import ConnectFailedException

from CMSMonitoring import __version__ as cmsmonitoring_version
from CMSMonitoring.Validator import validate_schema, Schemas

MIN_STOMP_PY_VERSION = (6, 1, 1)
MONIT_RESERVED_KEYS = ['type', '_id', 'producer', 'timestamp']

try:
    assert stomp.__version__ >= MIN_STOMP_PY_VERSION
except ImportError:
    print("No stomp module found or stomp.py version is not greater than 6.1.0")
    sys.exit(1)

# global object which holds CMS Monitoring schemas
_schemas = Schemas(update=3600, jsonschemas=False)
_local_schemas = None


def validate(doc, schema, logger):
    """
    Helper function to validate given document against a schema

    Schemas are searched for locally, or within the central CMSMonitoring schemas.
    The provided schema name is compared to full file names and to their base file names (without extension).

    Return a list of offending keys and a list of unknown keys, or None, None if no validation has been performed.
    """
    global _local_schemas
    if _local_schemas is None:
        # First time running, try to find the schema locally
        # Second time, _local_schemas will be a dictionary and this is skipped
        _local_schemas = {}
        if os.path.isfile(schema):
            try:
                _local_schemas[schema] = json.load(open(schema))
                msg = f"Successfully loaded local schema {schema} for validation"
                logger.warn(msg)
            except ValueError:
                msg = f"Local schema {schema} is not json compliant"
                logger.error(msg)
    if schema in _local_schemas:
        return validate_schema(_local_schemas[schema], doc, logger)
    else:
        for sch in _schemas.schemas():
            if schema in [sch, os.path.basename(sch).rsplit('.')[0]]:
                return validate_schema(_schemas.schemas()[sch], doc, logger)
    msg = f"Schema not found: '{schema}'"
    logger.error(msg)
    return [], []


class StompyListener(stomp.ConnectionListener):
    """Auxiliary listener class to fetch all possible states in the Stomp connection."""

    def __init__(self, logger=None):
        if logger:
            self.logger = logger
            # INFO level is very verbose, see: https://github.com/jasonrbriggs/stomp.py/issues/400
            # bump the log level to the next level to reduce verbosity. FIXME: eventually remove it!
            thisLogLevel = logging.getLogger("stomp.py").getEffectiveLevel()
            thisLogLevel = thisLogLevel if thisLogLevel == 50 else thisLogLevel + 10
            logging.getLogger("stomp.py").setLevel(thisLogLevel)
        else:
            logging.basicConfig(format='%(asctime)s- StompAMQ:%(levelname)s-%(message)s', datefmt='%Y-%m-%dT%H:%M:%S.%f%z',
                                level=logging.WARNING)
            self.logger = logging.getLogger('StompyListener')

    @staticmethod
    def safe_headers(headers):
        """Return stripped headers"""
        hdrs = dict(headers)
        for key in ['username', 'password', 'login', 'passcode']:
            if key in hdrs:
                hdrs[key] = 'xxx'
        return hdrs

    def on_connecting(self, host_and_port):
        """print debug message on_connecting"""
        self.logger.debug(f"on_connecting {str(host_and_port)}")

    def on_error(self, frame):
        """print debug message on_error"""
        self.logger.debug(f"received an error HEADERS: {str(self.safe_headers(frame.headers))}, "
                          f"BODY: {str(frame.body)} ...")

    def on_message(self, frame):
        """print debug message on_message"""
        self.logger.debug(f"on_message HEADERS: {str(self.safe_headers(frame.headers))}, BODY: {str(frame.body)} ...")

    def on_heartbeat(self):
        """print debug message on_heartbeat"""
        self.logger.debug('on_heartbeat')

    def on_send(self, frame):
        """print debug message on_send"""
        self.logger.debug(f"on_send HEADERS:{str(self.safe_headers(frame.headers))}, BODY: {str(frame.body)[:160]} ...")

    def on_connected(self, frame):
        """print debug message on_connected"""
        self.logger.debug(f"on_connected HEADERS: {str(self.safe_headers(frame.headers))}, BODY: {str(frame.body)} ...")

    def on_disconnected(self):
        """print debug message on_disconnected"""
        self.logger.debug('on_disconnected')

    def on_heartbeat_timeout(self):
        """print debug message on_heartbeat_timeout"""
        self.logger.debug('on_heartbeat_timeout')

    def on_before_message(self, frame):
        """print debug message on_before_message"""
        self.logger.debug(f"on_before_message HEADERS: {str(self.safe_headers(frame.headers))}, "
                          f"BODY: {str(frame.body)} ...")

        return frame


def broker_ips(host, port):
    """Return broker IP addresses from provide host name"""
    addr = []
    for item in socket.getaddrinfo(host, int(port)):
        # each item is (family, socktype, proto, canonname, sockaddr) tuple, so we tack 4th element sockaddr
        # for IPv4 sockaddr is (address, port)
        # for IPv6 sockaddr is (address, port, flow info, scope id)
        # we are interested only in address
        addr.append(item[4][0])
    return addr


# noinspection GrazieInspection
class StompAMQ7(object):
    """
    Class to generate and send notifications to a given Stomp broker and a given topic.

    :param username: The username to connect to the broker.
    :param password: The password to connect to the broker.
    :param producer: The 'producer' field in the notification header
    :param topic: The topic to be used on the broker
    :param validation_schema: schema to use for validation (filename of a valid json file).
        If 'None', skip any validation. Look for schema files locally, then in 'schemas/'
        folder in CMSMonitoring package or in folder defined in 'CMSMONITORING_SCHEMAS'
        environmental variable.
    :param host_and_ports: The hosts and ports list of the brokers.
        E.g.: [('cms-test-mb.cern.ch', 61313)]
    :param cert: path to certificate file
    :param key: path to key file
    :param loglevel: logging level, default is logging.WARNING
    :param timeout_interval: provides timeout interval to failed broker
    :param ipv4_only: use ipv4 servers only
    :param send_timeout: heartbeat send timeout in milliseconds
    :param recv_timeout: heartbeat receive timeout in milliseconds
    """

    # Version number is to be added in header
    _version = cmsmonitoring_version

    def __init__(self,
                 username,
                 password,
                 producer,
                 topic,
                 validation_schema,
                 host_and_ports=None,
                 logger=None,
                 cert=None,
                 key=None,
                 loglevel=logging.WARNING,
                 timeout_interval=600,
                 ipv4_only=True,
                 send_timeout=4000,
                 recv_timeout=4000):
        if logger:
            self.logger = logger
            # INFO level is very verbose, see: https://github.com/jasonrbriggs/stomp.py/issues/400
            # bump the log level to the next level to reduce verbosity. FIXME: eventually remove it!
            thisLogLevel = logging.getLogger("stomp.py").getEffectiveLevel()
            thisLogLevel = thisLogLevel if thisLogLevel == 50 else thisLogLevel + 10
            logging.getLogger("stomp.py").setLevel(thisLogLevel)
        else:
            # Set logger
            logging.basicConfig(format="%(asctime)s.%(msecs)03dZ [%(levelname)s] %(filename)s:%(lineno)d %(message)s ",
                                datefmt="%Y-%m-%dT%H:%M:%S",
                                level=loglevel)
            logging.Formatter.converter = gmtime
            self.logger = logging.getLogger('StompAMQ')

        self._username, self._password, self._producer, self._topic = username, password, producer, topic
        self._host_and_ports = host_and_ports or [('cms-test-mb.cern.ch', 61323)]
        self.ip_and_ports = []
        try:
            self.logger.info(f"host and ports: {repr(host_and_ports)}")
            if isinstance(host_and_ports, list):
                for host, port in host_and_ports:
                    for ipaddr in broker_ips(host, port):
                        if (ipaddr, port) not in self.ip_and_ports:
                            if ipv4_only:
                                if ipaddr.find(':') == -1:
                                    self.ip_and_ports.append((ipaddr, port))
                            else:
                                self.ip_and_ports.append((ipaddr, port))
                self.logger.info(f"resolver: {self.ip_and_ports}")
        except Exception as exp:
            self.logger.error(f"Unable to resolve host_and_ports: {str(exp)}")

        self._cert, self._key = cert, key

        # If cert and key pem files provided, use ssl.
        self._use_ssl = True if key and cert else False
        self.validation_schema = validation_schema

        if self.validation_schema is None:
            self.logger.debug('No document validation performed!')

        self.connections = []
        if self.ip_and_ports:
            for idx in range(len(self.ip_and_ports)):
                host_and_ports = [self.ip_and_ports[idx]]
                try:
                    conn = stomp.Connection(host_and_ports=host_and_ports, heartbeats=(send_timeout, recv_timeout))
                    desc = f"host: {host_and_ports}"
                    if self._use_ssl:
                        conn.set_ssl(for_hosts=host_and_ports, key_file=self._key, cert_file=self._cert)
                        desc = f"host: {host_and_ports}, ckey: {self._key}, cert: {self._cert}"
                        self.logger.info(desc)
                    self.connections.append((conn, desc))
                except Exception as exp:
                    msg = "Fail to connect to message broker\n"
                    msg += f"Host: {str(host_and_ports)}\n" + f"Error: {str(exp)}"
                    self.logger.warning(msg)
        else:
            try:
                conn = stomp.Connection(host_and_ports=self._host_and_ports, heartbeats=(send_timeout, recv_timeout))
                desc = f"host: {self._host_and_ports}"
                if self._use_ssl:
                    conn.set_ssl(for_hosts=self._host_and_ports, key_file=self._key, cert_file=self._cert)
                    desc = f"host: {host_and_ports}, ckey: {self._key}, cert: {self._cert}"
                self.connections.append((conn, desc))
            except Exception as exp:
                msg = "Fail to connect to message broker\n"
                msg += f"Host: {str(host_and_ports)}\n" + f"Error: {str(exp)}"
                self.logger.warning(msg)
        self.timeouts = {}
        self.timeout_interval = timeout_interval

    def connect(self):
        """Connect to the brokers"""
        available_connections = []
        for conn, desc in self.connections:
            # check if we already connected, if so make it available and proceed
            if conn.is_connected():
                available_connections.append(conn)
                continue

            conn.set_listener('StompyListener', StompyListener(self.logger))
            # check if our connection failed before, if so we'll wait until timeout_interval is passed
            if conn in self.timeouts and abs(self.timeouts[conn] - time.time()) < self.timeout_interval:
                continue
            try:
                # If cert/key are used, ignore username and password
                if self._use_ssl:
                    conn.set_ssl(for_hosts=self._host_and_ports, key_file=self._key, cert_file=self._cert)
                    conn.connect(wait=True)
                else:
                    conn.connect(username=self._username, passcode=self._password, wait=True)
                available_connections.append(conn)
                # we succeed to connect to broker, remove any record in timeout dict
                if conn in self.timeouts:
                    del self.timeouts[conn]
                self.logger.debug(f"Connection to {repr(self._host_and_ports)} is successful")
            except Exception as exc:
                traceback.print_exc()
                self.logger.error(f"Connection to {desc} failed, error: {str(exc)}")
                # record that our connection has failed
                self.timeouts[conn] = time.time()
                if conn.is_connected():
                    try:
                        conn.disconnect()
                    except (ConnectFailedException, Exception):
                        traceback.print_exc()

        if not available_connections:
            return None

        # return random connection
        idx = random.randint(0, len(available_connections) - 1)
        self.logger.debug(f"available connections {len(available_connections)}, con_id {idx}")
        return available_connections[idx]

    def disconnect(self):
        """Disconnect from brokers"""
        for conn, _ in self.connections:
            if conn.is_connected():
                try:
                    conn.disconnect()
                except ConnectionError:
                    traceback.print_exc()

    def send(self, data):
        """
        Connect to the stomp host and send a single notification (or a list of notifications).

        :param data: Either a single notification (as returned by `make_notification`) or a list of such

        :return: a list of notification bodies that failed to send
        """
        # If only a single notification, put it in a list
        if isinstance(data, dict) and 'body' in data:
            data = [data]

        failed_notifications = []
        try:
            conn = self.connect()  # provide random connection to brokers
            if conn:
                for notification in data:
                    result = self._send_single(conn, notification)
                    if result:
                        failed_notifications.append(result)
        except Exception as e:
            logging.warning("Send failed with exception: %s", str(e))
        # Do not use conn.disconnect() in version 6.1.1<=, <=7.0.0. It produces unnecessary socket warning

        if failed_notifications:
            self.logger.warning(f"Failed to send to {repr(self._host_and_ports)} {len(failed_notifications)} docs "
                                f"out of {len(data)}")

        return failed_notifications

    def _send_single(self, conn, notification):
        """
        Send a single notification to `conn`

        :param conn: An already connected stomp.Connection
        :param notification: A dictionary as returned by `make_notification`

        :return: The notification body in case of failure, or else None
        """
        failed_notification = None
        body = None
        try:
            body = notification.pop('body')
            conn.send(destination=self._topic,
                      headers=notification,
                      body=json.dumps(body),
                      ack='auto')
            self.logger.debug(f"Notification {str(notification)} sent")
        except Exception as exc:
            self.logger.error(f"Notification: {str(notification)} (type={type(notification)}) not send: {str(exc)}")
            failed_notification = body or {}
        return failed_notification

    def send_as_tx(self,
                   data,
                   doc_ype=None,
                   ts=None,
                   metadata=None,
                   data_subfield=None,
                   schema=None,
                   drop_offending_keys=False,
                   drop_unknown_keys=False):
        """
        Connect to the stomp host and send notifications as transaction.

        :param data: Either a single dictionary or a list of dictionaries.
        :param doc_ype: document type for metadata. MONIT document type
        :param ts: timestamp to be added to metadata. Set as time.time() by default
        :param metadata: dictionary of user metadata to be added. (Updates generic metadata.)
        :param data_subfield: field name to use for the actual data. If none, the data is put directly in the body.
        :param schema: Use this schema template to validate the payload. This should be
               the name of a json file looked for locally, or inside the folder defined
               in the 'CMSMONITORING_SCHEMAS' environment variable, or one of the defaults
               provided with the CMSMonitoring package. If 'None', the schema from the
               StompAMQ instance is applied. If that is also 'None', no validation is
               performed.
        :param drop_offending_keys: Drop keys that failed validation from the notification
        :param drop_unknown_keys: Drop keys not present in schema from the notification

        :return: a list of notification bodies that failed to send
        """
        if not isinstance(data, list):
            data = [data]

        conn = self.connect()  # provide random connection to brokers
        if conn:
            txid = None
            subscription_id = None

            # Subscribe
            try:
                # Unique id to identify subscription: https://stomp.github.io/stomp-specification-1.2.html#SUBSCRIBE
                subscription_id = str(uuid4())
                conn.subscribe(destination=self._topic, id=subscription_id)
                self.logger.debug(f"Subscription id: {subscription_id}")
            except Exception as e:
                self.logger.error(f"Subscription error. topic: {self._topic}, subscr id: {subscription_id}.", str(e))

            # Connect
            try:
                txid = conn.begin()  # returns transaction id
                self.logger.info(f"Transaction starting, transaction id: {txid}, size: {len(data)}")
            except Exception as e:
                self.logger.error(f"Connection error.", str(e))

            # Send
            try:
                for msg in data:
                    notif, off_keys, unk_keys = self.make_notification(payload=msg,
                                                                       doc_type=doc_ype,
                                                                       producer=self._producer,
                                                                       data_subfield=data_subfield,
                                                                       ts=ts,
                                                                       metadata=metadata,
                                                                       schema=schema,
                                                                       drop_offending_keys=drop_offending_keys,
                                                                       drop_unknown_keys=drop_unknown_keys)
                    self.logger.debug(f"Offending keys: {off_keys}, unknown keys: {unk_keys}")

                    # Dump body to json
                    body = None
                    try:
                        body = notif.pop('body')
                        body = json.dumps(body)
                    except TypeError:
                        self.logger.error("Unable to serialize the object: %s", body)

                    # Stomp conn.send, not self.send
                    conn.send(destination=self._topic,
                              headers=notif,
                              body=json.dumps(body),
                              ack='auto',
                              transaction=txid)
            except Exception as e:
                self.logger.error(f"Failed to send to {repr(self._host_and_ports)}; transaction id: {txid}, "
                                  f"data size {len(data)}, last msg of data {data[-1]}.", str(e))

            # Commit transaction
            try:
                conn.commit(txid)
                self.logger.info(f"Transaction finished, transaction id: {txid}, size: {len(data)}")
            except Exception as e:
                self.logger.error(f"Transaction commit failed. {repr(self._host_and_ports)}; transaction id: {txid}, "
                                  f"data size {len(data)}, last msg of data {data[-1]}.", str(e))

    def make_notification(self,
                          payload,
                          doc_type=None,
                          doc_id=None,
                          producer=None,
                          ts=None,
                          metadata=None,
                          data_subfield=None,
                          schema=None,
                          drop_offending_keys=False,
                          drop_unknown_keys=False):

        """
        Produce a notification from a single payload, adding the necessary
        headers and metadata. Generic metadata is generated to include a
        timestamp, producer name, document id, and a unique id. User can
        pass additional metadata which updates the generic metadata.

        If payload already contains a metadata field, it is overwritten.

        :param payload: Actual data.
        :param doc_type: document type for metadata. MONIT document type
        :param doc_id: document id representing the notification. If none provided, a unique id is created.
        :param producer: The notification producer name, taken from the StompAMQ instance producer name by default.
        :param ts: timestamp to be added to metadata. Set as time.time() by default
        :param metadata: dictionary of user metadata to be added. (Updates generic metadata.)
        :param data_subfield: field name to use for the actual data. If none, the data is put directly in the body.
        :param schema: Use this schema template to validate the payload. This should be
               the name of a json file looked for locally, or inside the folder defined
               in the 'CMSMONITORING_SCHEMAS' environment variable, or one of the defaults
               provided with the CMSMonitoring package. If 'None', the schema from the
               StompAMQ instance is applied. If that is also 'None', no validation is performed.
        :param drop_offending_keys: Drop keys that failed validation from the notification
        :param drop_unknown_keys: Drop keys not present in schema from the notification

        :return: a single notifications with the proper headers and metadata and lists of
               offending and unknown keys
        """
        # Do not allow MONIT reserved keys in given metadata
        if metadata is None:
            metadata = {}
        if isinstance(metadata, dict) and any(item in metadata for item in MONIT_RESERVED_KEYS):
            logging.error(f"metadata includes one of MONIT reserved key: {MONIT_RESERVED_KEYS}. "
                          f"Please use method params for reserved keys: producer, doc_id, doc_type, ts")
            sys.exit(1)

        # If timestamp is not provided, current time will be used
        ts = ts or int(time.time() * 1000)  # MONIT requires type msec
        # Get producer from function itself or StompAMQ7 instance
        producer = producer or self._producer
        # Set producer and timestamp
        metadata.update({'producer': producer, 'timestamp': ts})

        # If value is None, key should not be exist in the dict
        _ = doc_id and metadata.update({'_id': doc_id})
        _ = doc_type and metadata.update({'type': doc_type})

        # Validate the payload
        schema = schema or self.validation_schema
        offending_keys, unknown_keys = [], []
        if schema:
            offending_keys, unknown_keys = validate(payload, schema, self.logger)
            if offending_keys:
                msg = f"Document {doc_id} conflicts with schema '{schema}'"
                self.logger.warning(msg)
                if drop_offending_keys:
                    for key in offending_keys:
                        payload.pop(key)
            if unknown_keys:
                msg = f"Document {doc_id} contains keys not present in schema '{schema}'"
                self.logger.warning(msg)
                if drop_unknown_keys:
                    for key in unknown_keys:
                        payload.pop(key)

        headers = {'version': self._version, 'producer': producer}
        body = {}
        if data_subfield:
            body[data_subfield] = payload
        else:
            body = payload

        notification = {}
        notification.update(headers)
        notification['body'] = body
        if metadata:
            notification['body']['metadata'] = metadata

        return notification, offending_keys, unknown_keys
