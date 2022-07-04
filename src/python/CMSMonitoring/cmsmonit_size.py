#!/usr/bin/env python
# -*- coding: utf-8 -*-
# pylint: disable=
"""
File       : cms_hdfs_size.py
Author     : Valentin Kuznetsov <vkuznet AT gmail dot com>
Description: 
"""

# system modules
import os
import json
import time
import argparse

try:
    from CMSMonitoring.StompAMQ import StompAMQ
except ImportError:
    StompAMQ = None


class OptionParser:
    def __init__(self):
        """User based option parser"""
        self.parser = argparse.ArgumentParser(prog='PROG')
        self.parser.add_argument("--fin", action="store",
                                 dest="fin", default="", help="Input file")
        self.parser.add_argument("--fout", action="store",
                                 dest="fout", default="", help="Output file")
        self.parser.add_argument("--token", action="store",
                                 dest="token", default="", help="Token file to use for contacting ES MONIT")
        self.parser.add_argument("--amq", action="store",
                                 dest="amq", default="", help="credentials file for StompAMQ injection")
        self.parser.add_argument("--verbose", action="store_true",
                                 dest="verbose", default=False, help="verbose output")


def credentials(fname=None):
    """Read credentials from MONIT_BROKER environment"""
    if not fname:
        fname = os.environ.get('MONIT_BROKER', '')
    if not os.path.isfile(fname):
        raise Exception("Unable to locate MONIT credentials, please setup MONIT_BROKER")
    with open(fname, 'r') as istream:
        data = json.load(istream)
    return data


def get_size(output):
    val = float(output[0])
    att = output[1]
    if att == 'K':
        val = val * 1024
    elif att == 'M':
        val = val * 1024 * 1024
    elif att == 'G':
        val = val * 1024 * 1024 * 1024
    elif att == 'T':
        val = val * 1024 * 1024 * 1024 * 1024
    elif att == 'P':
        val = val * 1024 * 1024 * 1024 * 1024 * 1024
    return val


def run_esindex(cmd):
    """run given command"""
    output = os.popen(cmd).read()

    index, size = output.split()
    return size, index


def run_hdfs(cmd):
    """run given command"""
    output = os.popen(cmd).read().replace('\n', '')
    size = get_size(output.split())
    return size, output


def hdfs(fin, fout, token, amq, verbose):
    """perform HDFS scan"""
    out = []
    data = json.load(open(fin))
    cmd = "hadoop fs -du -h -s %s"
    # path = "hdfs:///path"
    for desc, path in data.items():
        size, output = run_hdfs(cmd % path)
        rec = {'name': desc, 'path': path, 'size': size, 'type': 'hdfs'}
        if verbose:
            print(desc, path, size, output)
        out.append(rec)

    # get monit ES info
    if token and os.path.exists(token):
        cmd = 'monit -token %s -query="stats"' % token
        output = os.popen(cmd).readlines()
        for line in output:
            index, size = line.replace('\n', '').split()
            rec = {'name': index, 'size': float(size), 'path': '', 'type': 'elasticsearch'}
            if verbose:
                print(index, size, line)
            out.append(rec)
    if amq:
        creds = credentials(amq)
        host, port = creds['host_and_ports'].split(':')
        port = int(port)
        producer = creds['producer']
        topic = creds['topic']
        username = creds['username']
        password = creds['password']
        if verbose:
            print("producer: {}, topic {}".format(producer, topic))
            print("host: {}, port: {}".format(host, port))
        try:
            # create instance of StompAMQ object with your credentials
            mgr = StompAMQ(username, password,
                           producer, topic,
                           validation_schema=None, host_and_ports=[(host, port)])
            # loop over your document records and create notification documents
            # we will send to MONIT
            data = []
            for doc in out:
                # every document should be hash id
                hid = doc.get("hash", 1)  # replace this line with your hash id generation
                tstamp = int(time.time()) * 1000
                producer = creds["producer"]
                notification, _, _ = \
                    mgr.make_notification(doc, hid, producer=producer, ts=tstamp, dataSubfield="")
                data.append(notification)

            # send our data to MONIT
            results = mgr.send(data)
            print("AMQ submission results", results)
        except Exception as exc:
            print("Fail to send data to AMQ", str(exc))
    else:
        if fout:
            with open(fout, 'w') as ostream:
                ostream.write(json.dumps(out))
        else:
            print(json.dumps(out))


def main():
    """Main function"""
    optmgr = OptionParser()
    opts = optmgr.parser.parse_args()
    hdfs(opts.fin, opts.fout, opts.token, opts.amq, opts.verbose)


if __name__ == '__main__':
    main()
