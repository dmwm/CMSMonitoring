#!/usr/bin/env python
#-*- coding: utf-8 -*-
#pylint: disable=C0301,R0903,R0912,W0703,C0103,R0914,W0603,R0902,R0913
"""
Basic interface to CERN ActiveMQ via stomp
"""
from __future__ import print_function
from __future__ import division

import os
import json
import time
import socket
import random
import logging
try:
    import stomp
except ImportError:
    print("No stomp module found")
from uuid import uuid4

from CMSMonitoring.Validator import validate_schema, Schemas

# global object which holds CMS Monitoring schemas
_schemas = Schemas(update=3600, jsonschemas=False)
_local_schemas = None

def validate(doc, schema, logger):
    """
    Helper function to validate given document against a schema

    Schemas are searched for locally, or within the central CMSMonitoring
    schemas. The provided schema name is compared to full file names and
    to their base file names (without extension).

    Return a list of offending keys and a list of unknown keys, or None, None
    if no validation has been performed.
    """
    global _local_schemas
    if _local_schemas is None:
        # First time running, try to find the schema locally
        # Second time, _local_schemas will be a dictionary and this is skipped
        _local_schemas = {}
        if os.path.isfile(schema):
            try:
                _local_schemas[schema] = json.load(open(schema))
                msg = 'Successfully loaded local schema {} for validation'.format(schema)
                logger.warn(msg)
            except ValueError:
                msg = 'Local schema {} is not json compliant'.format(schema)
                logger.error(msg)

    if schema in _local_schemas:
        return validate_schema(_local_schemas[schema], doc, logger)

    else:
        for sch in _schemas.schemas():
            if schema in [sch, os.path.basename(sch).rsplit('.')[0]]:
                return validate_schema(_schemas.schemas()[sch], doc, logger)

    msg = "Schema not found: '{}'".format(schema)
    logger.error(msg)
    return None, None

class StompyListener(object):
    """
    Auxiliar listener class to fetch all possible states in the Stomp
    connection.
    """
    def __init__(self, logger=None):
        logging.basicConfig(level=logging.debug)
        self.logger = logger if logger else logging.getLogger('StompyListener')

    def safe_headers(self, headers):
        "Return stripped headers"
        hdrs = dict(headers)
        for key in ['username', 'password', 'login', 'passcode']:
            if key in hdrs:
                hdrs[key] = 'xxx'
        return hdrs

    def on_connecting(self, host_and_port):
        "print debug message on_connecting"
        self.logger.debug('on_connecting %s', str(host_and_port))

    def on_error(self, headers, message):
        "print debug message on_error"
        self.logger.debug('received an error HEADERS: %s, MESSAGE: %s', \
                str(self.safe_headers(headers)), str(message))

    def on_message(self, headers, body):
        "print debug message on_message"
        self.logger.debug('on_message HEADERS: %s BODY: %s', \
                str(self.safe_headers(headers)), str(body))

    def on_heartbeat(self):
        "print debug message on_heartbeat"
        self.logger.debug('on_heartbeat')

    def on_send(self, frame):
        "print debug message on_send"
        self.logger.debug('on_send HEADERS: %s, BODY: %s ...', \
                str(self.safe_headers(frame.headers)), str(frame.body)[:160])

    def on_connected(self, headers, body):
        "print debug message on_connected"
        self.logger.debug('on_connected HEADERS: %s, BODY: %s', \
                str(self.safe_headers(headers)), str(body))

    def on_disconnected(self):
        "print debug message on_disconnected"
        self.logger.debug('on_disconnected')

    def on_heartbeat_timeout(self):
        "print debug message on_heartbeat_timeout"
        self.logger.debug('on_heartbeat_timeout')

    def on_before_message(self, headers, body):
        "print debug message on_before_message"
        self.logger.debug('on_before_message HEADERS: %s, BODY: %s', \
                str(self.safe_headers(headers)), str(body))

        return (headers, body)

def broker_ips(host, port):
    "Return broker IP addresses from provide host name"
    addr = []
    for item in socket.getaddrinfo(host, int(port)):
        # each item is (family, socktype, proto, canonname, sockaddr) tuple
        # so we tack 4th element sockaddr
        # for IPv4 sockaddr is (address, port)
        # for IPv6 sockaddr is (address, port, flow info, scope id)
        # we are interested only in address
        addr.append(item[4][0])
    return addr

class StompAMQ(object):
    """
    Class to generate and send notifications to a given Stomp broker
    and a given topic.

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
    :param validation_loglevel: logging level to use for validation feedback
    :param timeout_interval: provides timeout interval to failed broker
    :param ipv4_only: use ipv4 servers only
    """

    # Version number to be added in header
    _version = '0.3'

    def __init__(self, username, password, producer, topic, validation_schema,
                 host_and_ports=None, logger=None, cert=None, key=None,
                 validation_loglevel=logging.WARNING,
                 timeout_interval=600, ipv4_only=False):
        self._username = username
        self._password = password
        self._producer = producer
        self._topic = topic
        logging.basicConfig(level=validation_loglevel)
        self.logger = logger if logger else logging.getLogger('StompAMQ')
        self._host_and_ports = host_and_ports or [('cms-test-mb.cern.ch', 61313)]
        self.ip_and_ports = []
        try:
            self.logger.info("host and ports: %s", repr(host_and_ports))
            if isinstance(host_and_ports, list):
                for host, port in host_and_ports:
                    for ipaddr in broker_ips(host, port):
                        if (ipaddr, port) not in self.ip_and_ports:
                            if ipv4_only:
                                if ipaddr.find(':') == -1:
                                    self.ip_and_ports.append((ipaddr, port))
                            else:
                                self.ip_and_ports.append((ipaddr, port))
                self.logger.info("resolver: %s", self.ip_and_ports)
        except Exception as exp:
            self.logger.warn("unable to resolve host_and_ports: %s", str(exp))
        self._cert = cert
        self._key = key
        self._use_ssl = True if key and cert else False
        # silence the INFO log records from the stomp library, until this issue gets fixed:
        # https://github.com/jasonrbriggs/stomp.py/issues/226
        logging.getLogger("stomp.py").setLevel(logging.WARNING)
        self.validation_schema = validation_schema
        self.validation_loglevel = validation_loglevel

        if self.validation_schema is None:
            self.logger.warn('No document validation performed!')

        self.connections = []
        if self.ip_and_ports:
            for idx in range(len(self.ip_and_ports)):
                host_and_ports = [self.ip_and_ports[idx]]
                try:
                    conn = stomp.Connection(host_and_ports=host_and_ports)
                    if self._use_ssl:
                        # This requires stomp >= 4.1.15
                        conn.set_ssl(for_hosts=host_and_ports, \
                                key_file=self._key, cert_file=self._cert)
                    self.connections.append(conn)
                except Exception as exp:
                    msg = 'Fail to connect to message broker, error: %s' \
                            % str(exp)
                    self.logger.warn(msg)
        else:
            try:
                conn = stomp.Connection(host_and_ports=self._host_and_ports)
                if self._use_ssl:
                    # This requires stomp >= 4.1.15
                    conn.set_ssl(for_hosts=self._host_and_ports, \
                            key_file=self._key, cert_file=self._cert)
                self.connections.append(conn)
            except Exception as exp:
                msg = 'Fail to connect to message broker, error: %s' \
                        % str(exp)
                self.logger.warn(msg)
        self.timeouts = {}
        self.timeout_interval = timeout_interval

    def connect(self):
        "Connect to the brokers"
        available_connections = []
        for conn in self.connections:
            # check if we already connected, if so make it available and proceed
            if conn.is_connected():
                available_connections.append(conn)
                continue

            conn.set_listener('StompyListener', StompyListener(self.logger))
            # check if our connection failed before
            # if so we'll wait until timeout_interval is passed
            if conn in self.timeouts and \
                    abs(self.timeouts[conn] - time.time()) < self.timeout_interval:
                continue
            try:
                conn.start()
                # If cert/key are used, ignore username and password
                if self._use_ssl:
                    conn.connect(wait=True)
                else:
                    conn.connect(username=self._username, passcode=self._password, wait=True)
                available_connections.append(conn)
                # we succeed to connect to broker, remove any record in timeout dict
                if conn in self.timeouts:
                    del self.timeouts[conn]
                self.logger.debug("Connection to %s is successful", repr(self._host_and_ports))
            except stomp.exception.ConnectFailedException as exc:
                tstamp = time.strftime("%b %d %Y %H:%M:%S", time.gmtime())
                msg = "%s, connection to %s failed\n%s" \
                        % (tstamp, repr(self._host_and_ports), str(exc))
                msg += 'Available hosts: %s' % str(self.ip_and_ports)
                self.logger.error(msg)
                # record that our connection has failed
                self.timeouts[conn] = time.time()

        if not available_connections:
            return None

        # return random connection
        idx = random.randint(0, len(available_connections)-1)
        self.logger.debug("available connections %s, con_id %s", len(available_connections), idx)
        return available_connections[idx]

    def disconnect(self):
        "Disconnect from brokers"
        for conn in self.connections:
            if conn.is_connected():
                conn.disconnect()

    def send(self, data):
        """
        Connect to the stomp host and send a single notification
        (or a list of notifications).

        :param data: Either a single notification (as returned by
            `make_notification`) or a list of such.

        :return: a list of notification bodies that failed to send
        """
        # If only a single notification, put it in a list
        if isinstance(data, dict) and 'body' in data:
            data = [data]

        failedNotifications = []
        for notification in data:
            conn = self.connect() # provide random connection to brokers
            if conn:
                result = self._send_single(conn, notification)
                if result:
                    failedNotifications.append(result)

        self.disconnect() # disconnect all available connections to brokers

        if failedNotifications:
            self.logger.warning('Failed to send to %s %i docs out of %i', repr(self._host_and_ports),
                                len(failedNotifications), len(data))

        return failedNotifications

    def _send_single(self, conn, notification):
        """
        Send a single notification to `conn`

        :param conn: An already connected stomp.Connection
        :param notification: A dictionary as returned by `make_notification`

        :return: The notification body in case of failure, or else None
        """
        try:
            body = notification.pop('body')
            conn.send(destination=self._topic,
                      headers=notification,
                      body=json.dumps(body),
                      ack='auto')
            self.logger.debug('Notification %s sent', str(notification))
        except Exception as exc:
            self.logger.error('Notification: %s (type=%s) not send, error: %s', \
                    str(notification), type(notification), str(exc))
        return

    def make_notification(self, payload, docType, docId=None, \
            producer=None, ts=None, metadata=None, \
            dataSubfield="data", schema=None, \
            dropOffendingKeys=False, dropUnknownKeys=False):
        """
        Produce a notification from a single payload, adding the necessary
        headers and metadata. Generic metadata is generated to include a
        timestamp, producer name, document id, and a unique id. User can
        pass additional metadata which updates the generic metadata.

        If payload already contains a metadata field, it is overwritten.

        :param payload: Actual data.
        :param docType: document type for metadata.
        :param docId: document id representing the notification. If none provided,
               a unique id is created.
        :param producer: The notification producer name, taken from the StompAMQ
               instance producer name by default.
        :param ts: timestamp to be added to metadata. Set as time.time() by default
        :param metadata: dictionary of user metadata to be added. (Updates generic
               metadata.)
        :param dataSubfield: field name to use for the actual data. If none, the data
               is put directly in the body. Default is "data"
        :param schema: Use this schema template to validate the payload. This should be
               the name of a json file looked for locally, or inside the folder defined
               in the 'CMSMONITORING_SCHEMAS' environment variable, or one of the defaults
               provided with the CMSMonitoring package. If 'None', the schema from the
               StompAMQ instance is applied. If that is also 'None', no validation is
               performed.
        :param dropOffendingKeys: Drop keys that failed validation from the notification
        :param dropUnknownKeys: Drop keys not present in schema from the notification

        :return: a single notifications with the proper headers and metadata and lists of
               offending and unknown keys
        """
        producer = producer or self._producer
        umetadata = metadata or {}
        ts = ts or int(time.time())
        uuid = str(uuid4())
        docId = docId or uuid

        # Validate the payload
        schema = schema or self.validation_schema
        offending_keys, unknown_keys = [], []
        if schema:
            offending_keys, unknown_keys = validate(payload, schema, self.logger)
            if offending_keys:
                msg = "Document {} conflicts with schema '{}'".format(docId, schema)
                self.logger.warn(msg)
                if dropOffendingKeys:
                    for key in offending_keys:
                        payload.pop(key)

            if unknown_keys:
                msg = "Document {} contains keys not present in schema '{}'".format(docId, schema)
                self.logger.warn(msg)
                if dropUnknownKeys:
                    for key in unknown_keys:
                        payload.pop(key)

        headers = {'type': docType,
                   'version': self._version,
                   'producer': producer}

        metadata = {'timestamp': ts,
                    'producer': producer,
                    '_id': docId,
                    'uuid': uuid}
        metadata.update(umetadata)

        body = {}
        if dataSubfield:
            body[dataSubfield] = payload
        else:
            body.update(payload)
        body['metadata'] = metadata

        notification = {}
        notification.update(headers)
        notification['body'] = body

        return notification, offending_keys, unknown_keys
