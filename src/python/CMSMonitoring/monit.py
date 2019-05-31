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
import traceback

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
        desc += '   monit --dbname=monit_prod_wmagent --query="status:Available,sum:35967"\n'
        desc += '\n'
        desc += 'Example of querying ES DB via ES query:\n'
        desc += '   monit --dbname=monit_prod_wmagent --query=query.json\n'
        desc += '   where query.json is ES query, e.g.\n'
        desc += '   {"query":{"match":{"data.payload.status": "Available"}}}\n'
        desc += '   For ES query syntax see:\n'
        desc += '   https://www.elastic.co/guide/en/elasticsearch/reference/current/search-template.html\n'
        desc += '   https://www.elastic.co/guide/en/elasticsearch/reference/5.5/query-dsl-term-query.html\n'
        desc += '\n'
        desc += 'Example of querying InfluxDB:\n'
        desc += '   monit --dbname=monit_production_condor --query="SHOW TAG KEYS"\n'
        desc += '\n'
        desc += 'Token can be obtained at https://monit-grafana.cern.ch/org/apikeys\n'
        desc += 'For more info see: http://docs.grafana.org/http_api/auth/\n'
        desc += 'If token is not provided we will use default one'
        self.parser = argparse.ArgumentParser(prog='PROG', usage=desc)
        self.parser.add_argument("--dbid", action="store",
            dest="dbid", default=0, help="DB identifier")
        self.parser.add_argument("--dbname", action="store",
            dest="dbname", default="", help="DB name, e.g. monit_prod_wmagent")
        self.parser.add_argument("--query", action="store",
            dest="query", default="", help="DB query, JSON for ES and SQL for InfluxDB")
        token = '/afs/cern.ch/user/l/leggerf/cms/token.txt'
        self.parser.add_argument("--token", action="store",
            dest="token", default=token, help="User token")
        self.parser.add_argument("--url", action="store",
            dest="url", default="https://monit-grafana.cern.ch", help="MONIT URL")
        self.parser.add_argument("--idx", action="store",
            dest="idx", default=0, help="result index, default 0")
        self.parser.add_argument("--limit", action="store",
            dest="limit", default=10, help="result limit, default 10")
        self.parser.add_argument("--verbose", action="store",
            dest="verbose", default=0, help="verbose level, default 0")

DBDICT = {
        'CMSWEB': 8973,
        'cmsweb-k8s': 8488,
        'cmsweb-prometheus': 9041,
        'es-cms': 8983,
        'cms': 8983,
        'es-cms-cmssdt-relvals': 9015,
        'monit_condor': 7668,
        'monit_crab': 8978,
        'monit_eoscmsquotas': 9003,
        'monit_prod_condor': 8787,
        'monit_es_condor_2019': 9014,
        'monit_es_condor_tasks': 9039,
        'monit_es_toolsandint': 9040,
        'monit_idb_alarms': 8528,
        'monit_idb_availability': 9107,
        'monit_idb_cmsjm': 7731,
        'monit_idb_databases': 8530,
        'monit_idb_eos': 8531,
        'monit_idb_kpis': 8532,
        'monit_idb_monitoring': 9109,
        'monit_idb_rebus': 9115,
        'monit_idb_transfers': 8035,
        'monit_kpi': 8533,
        'monit_prod_tier0wmagent': 9113,
        'monit_prod_wmagent': 7617,
        'monit_prod_wmagentqa': 8429,
        'SCHEDD': 8980,
        'SI-condor': 8993,
        'TASKWORKER': 8981,
        'WMArchive': 7572
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
    try:
        return json.loads(response.text)
    except:
        print("response: %s" % response.text)
        traceback.print_exc()

def main():
    "Main function"
    optmgr  = OptionParser()
    opts = optmgr.parser.parse_args()
    dbid = int(opts.dbid)
    idx = int(opts.idx)
    limit = int(opts.limit)
    verbose = int(opts.verbose)
    dbname = opts.dbname
    query = opts.query
    token = open(opts.token).read().replace('\n', '') \
            if os.path.exists(opts.token) else opts.token
    if dbname == 'es-cms':
        dbname = 'cms'  #  https://github.com/dmwm/CMSMonitoring/issues/21
    results = run(opts.url, token, dbid, dbname, query, idx, limit, verbose)
    print(json.dumps(results))

if __name__ == '__main__':
    main()
