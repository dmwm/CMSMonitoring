#!/usr/bin/env python
#-*- coding: utf-8 -*-
#pylint: disable=
"""
File       : monit.py
Author     : Valentin Kuznetsov <vkuznet AT gmail dot com>
Description: CMS Monitoring MONIT query tool
"""

# system modules
import os
import sys
import json
import argparse
import requests

try:
    import urllib.request as ulib # python 3.X
    import urllib.parse as parser
except ImportError:
    import urllib2 as ulib # python 2.X
    import urllib as parser

class OptionParser():
    def __init__(self):
        "User based option parser"
        desc = "CMS MONIT query tool to query "
        desc += 'MONIT datatabases via ES/Influx DB queries or key:val pairs\n'
        desc += '\n'
        desc += 'Example of querying ES DB via key:val pairs:\n'
        desc += '   monit --token=$token --dbname=monit_prod_wmagent --query="status:Available,sum:35967"\n'
        desc += '\n'
        desc += 'Example of querying ES DB via ES query:\n'
        desc += '   monit --token=$token --dbname=monit_prod_wmagent --query=query.json\n'
        desc += '   where query.json is ES query, e.g.\n'
        desc += '   {"query":{"match":{"data.payload.status": "Available"}}}\n'
        desc += '   For ES query syntax see:\n'
        desc += '   https://www.elastic.co/guide/en/elasticsearch/reference/current/search-template.html\n'
        desc += '   https://www.elastic.co/guide/en/elasticsearch/reference/5.5/query-dsl-term-query.html\n'
        desc += '\n'
        desc += 'Example of querying InfluxDB:\n'
        desc += '   monit --token=$token --dbname=monit_production_condor --query="SHOW TAG KEYS"\n'
        self.parser = argparse.ArgumentParser(prog='PROG', usage=desc)
        self.parser.add_argument("--dbid", action="store",
            dest="dbid", default=0, help="DB identifier")
        self.parser.add_argument("--dbname", action="store",
            dest="dbname", default="", help="DB name, e.g. monit_prod_wmagent")
        self.parser.add_argument("--query", action="store",
            dest="query", default="", help="DB query, JSON for ES and SQL for InfluxDB")
        self.parser.add_argument("--token", action="store",
            dest="token", default="", help="User token")
        self.parser.add_argument("--url", action="store",
            dest="url", default="https://monit-grafana.cern.ch", help="MONIT URL")
        self.parser.add_argument("--idx", action="store",
            dest="idx", default=0, help="result index, default 0")
        self.parser.add_argument("--limit", action="store",
            dest="limit", default=10, help="result limit, default 10")
        self.parser.add_argument("--verbose", action="store",
            dest="verbose", default=0, help="verbose level, default 0")

DBDICT = {
    'monit_prod_wmagent': 7617,
    'monit_production_condor': 7731
}
def run(url, token, dbid, dbname, query, idx=0, limit=10, verbose=0):
    headers = {'Authorization': 'Bearer {}'.format(token)}
    dbid = dbid if dbid else DBDICT.get(dbname, 0)
    if not dbid:
        print('No dbid found, please provide either valid db name or dbid')
        sys.exit(1)
    if os.path.exists(query) or query.find(':') != -1: # ES query
        if not dbname.endswith('*'):
            dbname += '*'
        if os.path.isfile(query):
            ustr = open(query).read()
        else:
            ustr = query
        if query.find('search_type') == -1:
            qstr = {"search_type": "query_then_fetch", "ignore_unavailable": True, "index": [dbname]}
            try:
                qdict = json.loads(ustr)
            except ValueError:
                mlist = []
                for pair in ustr.split(','):
                    key, val = pair.split(':')
                    key = key.strip()
                    val = val.strip()
                    if not key.startswith('data.payload'):
                        key = 'data.payload.{}'.format(key)
                    mlist.append({'match':{key:val}})
                qdict = {"query":{"bool":{"must":mlist}}}
            if 'from' not in qdict:
                qdict.update({'from': idx})
            if 'size' not in qdict:
                qdict.update({'size': limit})
            query = '{}\n{}\n'.format(json.dumps(qstr), json.dumps(qdict))
        else:
            query = ustr
        return query_es(url, dbid, query, headers, verbose)
    # influxDB query
    return query_idb(url, dbid, dbname, query, headers, verbose)

def query_idb(base, dbid, dbname, query, headers, verbose=0):
    "Method to query InfluxDB"
    uri = base + '/api/datasources/proxy/{}/query?db={}&q={}'.format(dbid, dbname, query)
    response = requests.get(uri, headers=headers)
    return json.loads(response.text)

def query_es(base, dbid, query, headers, verbose=0):
    "Method to query ES DB"
    # see https://www.elastic.co/guide/en/elasticsearch/reference/5.5/search-multi-search.html
    headers.update({'Content-type': 'application/x-ndjson', 'Accept': 'application/json'})
    uri = base + '/api/datasources/proxy/{}/_msearch'.format(dbid)
    if verbose:
        print("Query ES: uri={} query={}".format(uri, query))
    response = requests.get(uri, data=query, headers=headers)
    return json.loads(response.text)

def main():
    "Main function"
    optmgr  = OptionParser()
    opts = optmgr.parser.parse_args()
    dbid = int(opts.dbid)
    idx = int(opts.idx)
    limit = int(opts.limit)
    verbose = int(opts.verbose)
    results = run(opts.url, opts.token, dbid, opts.dbname, opts.query, idx, limit, verbose)
    print(json.dumps(results))

if __name__ == '__main__':
    main()
