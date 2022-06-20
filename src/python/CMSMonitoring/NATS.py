#!/usr/bin/env python
# -*- coding: utf-8 -*-

"""
File       : NATS.py
Author     : Valentin Kuznetsov <vkuznet AT gmail dot com>
Description: python implementation of NATS publisher manager based on
- python3 implementation via asyncio module, see
  https://github.com/nats-io/nats.py
  https://pypi.org/project/asyncio-nats-client/
- external tool, via
  go get github.com/nats-io/go-nats-examples/tools/nats-pub

Basic usage instructions:
    
..doctest:

    data = [...]   # list of dicts
    topics = [...] # list of publisher topics, e.g. cms, T2
    attrs = [...]  # list of attributes to filter from data
    # server can be in a form of 'nats://host:port' or
    # nats with user/passwords 'nats://user:password@host:port'
    # tls  with user/password  'tls://user:password@host:port'
    server = 'nats://host:port'
    mgr = NATSManager(server, topics=topics, attrs=attrs, stdout=True)
    mgr.publish(data)
"""

import os
import random
import subprocess
import traceback
import asyncio
from nats.aio.client import Client as NATS


# python3 implementation of NATS python client
async def nats(server, subject, msg):
    """
    NATS client implemented via asyncio
    python3 implementation, see
    https://github.com/nats-io/nats.py
    """
    nc = NATS()
    try:
        await nc.connect(server, max_reconnect_attempts=3)
    except Exception as exp:
        print("failed to connect to server: error {}".format(str(exp)))
        traceback.print_exc()
        return
    if isinstance(msg, list):
        for item in msg:
            await nc.publish(subject, item)
    else:
        await nc.publish(subject, msg)
    await nc.close()


def nats_encoder(doc, sep='   '):
    """CMS NATS message encoder"""
    keys = sorted(doc.keys())
    msg = sep.join(['{}:{}'.format(k, doc[k]) for k in keys])
    return msg


def nats_decoder(msg, sep='   '):
    """CMS NATS message decoder"""
    rec = {}
    for pair in msg.split(sep):
        arr = pair.split(':')
        rec.update({arr[0]: arr[1]})
    return rec


def def_filter(doc, attrs=None):
    """Default filter function for given doc and attributes"""
    rec = {}
    if not attrs:
        yield doc
        return
    for attr in attrs:
        if attr in doc:
            rec[attr] = doc[attr]
    yield rec


def gen_topic(def_topic, key, val, sep=None):
    """Create proper subject topic for NATS"""
    if sep:
        val = str(val).replace(sep, '.')
        if val.startswith('.'):
            val = val[1:]
    else:
        val = str(val)
    topic = '{}.{}.{}'.format(def_topic, key, val)
    return topic


class NATSManager(object):
    """
    NATSManager provide python interface to NATS server. It accepts:
    :param server: server name
    :param topics: list of topics, i.e. where messages will be published
    :param attrs: list of attributes to select for publishing
    :param default_topic: default topic 'cms'
    :param stdout: instead of publishing print all messages on stdout, boolean
    :param cms_filter: name of cms filter function, by default `def_filter` will be used
    """

    def __init__(self, server=None, topics=None, attrs=None, sep='   ', default_topic='cms', stdout=False,
                 cms_filter=None):
        self.topics = topics
        self.server = []
        self.sep = sep
        for srv in server.split(','):
            self.server.append(srv)
        self.def_topic = default_topic
        self.stdout = stdout
        self.attrs = attrs
        self.cms_filter = cms_filter if cms_filter else def_filter

    def __repr__(self):
        return 'NATSManager@{}, topics={} def_topic={} attrs={} cms_filter={} stdout={}'.format(hex(id(self)),
                                                                                                self.topics,
                                                                                                self.def_topic,
                                                                                                self.attrs,
                                                                                                self.cms_filter,
                                                                                                self.stdout)

    def publish(self, data):
        """
        Publish given set of docs to specific topics.

        We use the following logic to publish messages. If this class is instantiated
        with set of topics the messages will be sent to those topics, otherwise
        we compose nested structure of topics starting from default_topic root (cms),
        e.g. cms.site.T3.US.Cornell, cms.task.taskName, cms.exitCode.123
        The sub-topics are based on given document attributes. Moreover, if
        document carry on system attribute key its value will be used as
        sub-root topic, e.g. if document has the form {..., "system":"dbs"},
        then topics will start from "cms.dbs" root where cms represent
        root topic, and dbs represents sub-root topic, respectively.
        """
        if isinstance(data, dict):
            data = [data]
        if not isinstance(data, list):
            print("NATS: invalid data type '%s', expect dict or list of dicts" % type(data))
        all_msgs = []
        mdict = {}
        subject = self.def_topic
        if not self.topics:
            for doc in data:
                srv = doc.get('system', '')
                if srv:
                    subject += '.' + srv
                    del doc['system']
                for rec in self.cms_filter(doc, self.attrs):
                    msg = nats_encoder(rec, self.sep)
                    for key, val in rec.items():
                        if key == 'exitCode' and str(val):
                            # be sure exitCode is converted to str
                            topic = gen_topic(subject, key, str(val))
                        elif key == 'site' and val != "":
                            # replace sites separator '_'
                            topic = gen_topic(subject, key, val, '_')
                        elif key == 'task' and val != "":
                            # replace task separator '/'
                            topic = gen_topic(subject, key, val, '/')
                        elif key == 'dataset' and val != "":
                            # replace task separator '/'
                            topic = gen_topic(subject, key, val, '/')
                        else:
                            if str(val):
                                # collect messages on individual topics
                                topic = gen_topic(subject, key, val)
                            else:
                                topic = ''
                        if topic:
                            mdict.setdefault(topic, []).append(msg)
                    all_msgs.append(msg)
            # send all messages from collected mdict
            for key, msgs in mdict.items():
                self.send(key, msgs)
            if not mdict:
                # send message to default topic
                self.send(self.def_topic, all_msgs)
            return
        for topic in self.topics:
            msgs = []
            for doc in data:
                for rec in self.cms_filter(doc, self.attrs):
                    msgs.append(nats_encoder(rec, self.sep))
            self.send(topic, msgs)

    def send_stdout(self, subject, msg):
        """send subject/msg to stdout"""
        if isinstance(msg, list):
            for item in msg:
                print("{}: {}".format(subject, item))
        else:
            print("{}: {}".format(subject, msg))

    def get_server(self):
        """Return random server from server pool"""
        if len(self.server) > 1:
            return self.server[random.randint(0, len(self.server) - 1)]
        return self.server[0]

    def send(self, subject, msg):
        """send given message to subject topic"""
        if self.stdout:
            self.send_stdout(subject, msg)
        else:
            try:
                server = self.get_server()
                # VK: we need to test initializing asyncio loop in ctor
                # and potentially avoid loop creation here
                loop = asyncio.get_event_loop()
                loop.run_until_complete(nats(server, subject, msg))
                loop.close()
            except Exception as exp:
                print("Failed to send docs to NATS, error: {}".format(str(exp)))


def nats_cmd(cmd, server, subject, msg):
    """NATS publisher via external nats cmd tool"""
    if not os.path.exists(cmd):
        print("Unable to locate publisher tool '{}' on local file system".format(cmd))
        return
    if not server:
        print("No NATS server...")
        return
    if isinstance(msg, list):
        for item in msg:
            pcmd = '{} -s {} {} "{}"'.format(cmd, server, subject, item)
            proc = subprocess.Popen(pcmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, shell=True, env=os.environ)
            proc.wait()
    else:
        pcmd = '{} -s {} {} "{}"'.format(cmd, server, subject, msg)
        proc = subprocess.Popen(pcmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, shell=True, env=os.environ)
        proc.wait()
        return proc.returncode


class NATSManagerCmd(NATSManager):
    """NATS manager based on external publisher tool"""

    def __init__(self, cmd, server=None, topics=None, attrs=None, sep='   ', default_topic='cms', stdout=False,
                 cms_filter=None):
        self.pub = cmd
        super(NATSManagerCmd, self).__init__(server, topics, attrs, sep, default_topic, stdout, cms_filter)

    def __repr__(self):
        return 'NATSManagerCmd@{}, topics={} def_topic={} attrs={} cms_filter={} stdout={}'.format(hex(id(self)),
                                                                                                   self.topics,
                                                                                                   self.def_topic,
                                                                                                   self.attrs,
                                                                                                   self.cms_filter,
                                                                                                   self.stdout)

    def send(self, subject, msg):
        """Call NATS function, user can pass either single message or list of messages"""
        if self.stdout:
            self.send_stdout(subject, msg)
        else:
            server = self.get_server()
            nats_cmd(self.pub, server, subject, msg)


def test():
    """Test function"""
    subject = 'cms'  # default topic
    doc = {'site': '1', 'attr': '1'}
    sep = '   '
    msg = nats_encoder(doc, sep)
    rec = nats_decoder(msg)
    nmsg = nats_encoder(rec)
    assert msg == nmsg
    print('--- encoding test OK ---')

    data = [{'site': 'T3_US_Test', 'campaign': 'campaign-%s' % str(i),
             'task': '/task%s/part%s' % (i, i), 'system': 'system',
             'exitCode': i} for i in range(2)]
    print('input data: {}'.format(data))
    server = '127.0.0.1'
    attrs = ['site', 'campaign', 'task', 'exitCode']
    mgr = NATSManager(server, attrs=attrs, stdout=True)
    print('\n--- no topic test ---')
    print(mgr)
    mgr.publish(data)

    # check new default end-point
    mgr = NATSManager(server, default_topic='cms.dbs', attrs=attrs, stdout=True)
    print('\n--- custom default end-point test ---')
    print(mgr)
    mgr.publish(data)

    # create new manager with topics
    topics = ['test-topic']
    mgr = NATSManager(server, topics=topics, attrs=attrs, stdout=True)
    print('\n--- topics test ---')
    print(mgr)
    mgr.publish(data)

    # create new manager with custom filter and topics
    def custom_filter(doc, attrs=None):
        "Custom filter function for given doc and attributes"
        rec = {}
        for attr in attrs:
            if attr in doc:
                rec[attr] = doc[attr]
        rec['enriched'] = True
        yield rec

    mgr = NATSManager(server, topics=topics, attrs=attrs, cms_filter=custom_filter, stdout=True)
    print('\n--- custom filter test ---')
    print(mgr)
    mgr.publish(data)


if __name__ == '__main__':
    test()
