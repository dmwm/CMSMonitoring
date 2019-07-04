#!/usr/bin/env python
#-*- coding: utf-8 -*-
#pylint: disable=
"""
File       : test_brokers.py
Author     : Valentin Kuznetsov <vkuznet AT gmail dot com>
Description: 
"""

# system modules
import os
import sys
import json
import logging
import argparse

from CMSMonitoring.StompAMQ import StompAMQ

class OptionParser():
    def __init__(self):
        "User based option parser"
        self.parser = argparse.ArgumentParser(prog='PROG')
        self.parser.add_argument("--broker-params", action="store",
            dest="broker_params", default="", help="Input file with broker params credentions")
        self.parser.add_argument("--ipv4", action="store_true",
            dest="ipv4", default=False, help="Use ipv4")
        self.parser.add_argument("--ckey", action="store",
            dest="ckey", default="", help="ckey")
        self.parser.add_argument("--cert", action="store",
            dest="cert", default="", help="cert")
        self.parser.add_argument("--verbose", action="store_true",
            dest="verbose", default=False, help="verbose output")

def credentials(fname=None):
    "Read credentials from CMS_BROKER environment"
    if  not fname:
        fname = os.environ.get('CMS_BROKER', '')
    if  not os.path.isfile(fname):
        return {}
    with open(fname, 'r') as istream:
        data = json.load(istream)
    return data

def test(fname, ipv4, ckey, cert, verbose):
    level = logging.DEBUG if verbose else logging.INFO
    logging.basicConfig(level=level)
    logger = logging.getLogger('TEST')
    creds = credentials(fname)
    host, port = creds['host_and_ports'].split(':')
    port = int(port)
    amq1 = StompAMQ(creds['username'], creds['password'],
            creds['producer'], creds['topic'],
            validation_schema = None,
            logger = logger, ipv4_only=ipv4,
            host_and_ports = [('cms-test-mb.cern.ch', 61313)])
    amq2 = StompAMQ(creds['username'], creds['password'],
            creds['producer'], creds['topic'],
            validation_schema = None,
            logger = logger,
            host_and_ports = [('cms-test-mb.cern.ch', 61323)],
            cert=cert, key=ckey, ipv4_only=ipv4)
    docs = [{'test':i, 'hash': i} for i in range(10)]
    data = []
    for doc in docs:
        doc_type = 'test'
        notification, okeys, ukeys = amq1.make_notification(doc, doc_type)
        data.append(notification)
    print("### Sending data with AMQ (user/pswd)")
    results = amq1.send(data)
    if results:
        print("### failed docs from AMQ %s" % len(results))
        if verbose:
            for doc in results:
                print(doc)
    print("### Sending data with AMQ (ckey/cert)")
    results = amq2.send(data)
    if results:
        print("### failed docs from AMQ %s" % len(results))
        if verbose:
            for doc in results:
                print(doc)

def main():
    "Main function"
    optmgr  = OptionParser()
    opts = optmgr.parser.parse_args()
    test(opts.broker_params, opts.ipv4, opts.ckey, opts.cert, opts.verbose)

if __name__ == '__main__':
    main()
