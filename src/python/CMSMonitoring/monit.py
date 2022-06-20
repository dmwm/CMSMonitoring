#!/usr/bin/env python
# -*- coding: utf-8 -*-
# pylint: disable=
"""
File       : monit.py
Author     : Valentin Kuznetsov <vkuznet AT gmail dot com>
Description: CMS Monitoring MONIT query tool
"""

# system modules
import os
import re
import sys
import json
import argparse
import requests
import traceback


class OptionParser:
    def __init__(self):
        """User based option parser"""
        desc = "CMS MONIT query tool to query "
        desc += "MONIT datatabases via ES/Influx DB queries or key:val pairs\n"
        desc += "The file containing the definition of the datasources is set using "
        desc += "the MONIT_DB_DICT_FILE environment variable\n"
        desc += "It will use by default the static/datasources.json file\n"
        desc += "\n"
        desc += "Example of querying ES DB via key:val pairs:\n"
        desc += '   monit --dbname=monit_prod_wmagent --query="status:Available,sum:35967"\n'
        desc += "\n"
        desc += "Example of querying ES DB via ES query:\n"
        desc += "   monit --dbname=monit_prod_wmagent --query=query.json\n"
        desc += "   where query.json is ES query, e.g.\n"
        desc += '   {"query":{"match":{"data.payload.status": "Available"}}}\n'
        desc += "   For ES query syntax see:\n"
        desc += "   https://www.elastic.co/guide/en/elasticsearch/reference/current/search-template.html\n"
        desc += "   https://www.elastic.co/guide/en/elasticsearch/reference/5.5/query-dsl-term-query.html\n"
        desc += "\n"
        desc += "Example of querying InfluxDB:\n"
        desc += '   monit --dbname=monit_condor --query="SHOW TAG KEYS"\n'
        desc += "\n"
        desc += "Token can be obtained at https://monit-grafana.cern.ch/org/apikeys\n"
        desc += "For more info see: http://docs.grafana.org/http_api/auth/\n"
        desc += "If token is not provided we will use default one"
        self.parser = argparse.ArgumentParser(prog="PROG", usage=desc)
        self.parser.add_argument(
            "--dbid", action="store", dest="dbid", default=0, help="DB identifier"
        )
        self.parser.add_argument(
            "--dbname",
            action="store",
            dest="dbname",
            default="",
            help="Name of the datasource in grafana, e.g. monit_condor, "
                 "you can list the datasources using the option --list_dbs",
        )
        self.parser.add_argument(
            "--query",
            action="store",
            dest="query",
            default="",
            help="DB query, JSON for ES and SQL for InfluxDB",
        )
        token = "/afs/cern.ch/user/l/leggerf/cms/token.txt"
        self.parser.add_argument(
            "--token", action="store", dest="token", default=token, help="User token"
        )
        self.parser.add_argument(
            "--url",
            action="store",
            dest="url",
            default="https://monit-grafana.cern.ch",
            help="MONIT URL",
        )
        self.parser.add_argument(
            "--idx",
            action="store",
            dest="idx",
            default=0,
            help="result index, default 0",
        )
        self.parser.add_argument(
            "--limit",
            action="store",
            dest="limit",
            default=10,
            help="result limit, default 10",
        )
        self.parser.add_argument(
            "--verbose",
            action="store",
            dest="verbose",
            default=0,
            help="verbose level, default 0",
        )
        self.parser.add_argument(
            "--list_dbs",
            action="store_true",
            dest="list_dbs",
            default=False,
            help="Prints the dbs list and exit",
        )


def run(url, token, dbid, dbname, query, idx=0, limit=10, verbose=0):
    if not query:
        print(
            "You need to specify the query (either a query string or a path to a file)"
        )
        sys.exit(1)
    headers = {"Authorization": "Bearer {}".format(token)}
    _db = None
    _tmp_dir = __get_db_dict()
    if not dbid:
        _db = _tmp_dir.get(dbname, {})
        dbid = _db.get("id")
    else:
        for key in _tmp_dir:
            if _tmp_dir[key].get("id") == dbid:
                _db = _tmp_dir[key]
                break
    if not dbid:
        print("No dbid found, please provide either valid db name or dbid")
        sys.exit(1)
    if os.path.exists(query) or (
        _db and query and _db.get("type") == "elasticsearch"
    ):  # ES query
        if os.path.isfile(query):
            ustr = open(query).read()
        else:
            ustr = query
        if ustr.find("search_type") == -1:
            qstr = {
                "search_type": "query_then_fetch",
                "ignore_unavailable": True,
                "index": __infer_index(_db, dbname),
            }
            try:
                qdict = json.loads(ustr)
            except ValueError:
                mlist = []
                for pair in ustr.split(","):
                    key, val = pair.split(":")
                    key = key.strip()
                    val = val.strip()
                    if not key.startswith("data.payload"):
                        key = "data.payload.{}".format(key)
                    mlist.append({"match": {key: val}})
                qdict = {"query": {"bool": {"must": mlist}}}
            if "from" not in qdict:
                qdict.update({"from": idx})
            if "size" not in qdict:
                qdict.update({"size": limit})
            query = "{}\n{}\n".format(json.dumps(qstr), json.dumps(qdict))
        else:
            query = ustr
        return query_es(url, dbid, query, headers, verbose)
    # influxDB query
    _db_name = _db.get("database") if _db else dbname
    return query_idb(url, dbid, _db_name, query, headers, verbose)


def query_idb(base, dbid, dbname, query, headers, verbose=0):
    """Method to query InfluxDB"""
    uri = base + "/api/datasources/proxy/{}/query?db={}&q={}".format(
        dbid, dbname, query
    )
    response = requests.get(uri, headers=headers)
    try:
        return json.loads(response.text)
    except Exception as e:
        print(e)
        print(response.text)
        sys.exit(1)


def query_es(base, dbid, query, headers, verbose=0):
    """Method to query ES DB"""
    # see https://www.elastic.co/guide/en/elasticsearch/reference/5.5/search-multi-search.html
    headers.update(
        {"Content-type": "application/x-ndjson", "Accept": "application/json"}
    )
    uri = base + "/api/datasources/proxy/{}/_msearch".format(dbid)
    if verbose:
        print("Query ES: uri={} query={}".format(uri, query))
    response = requests.get(uri, data=query, headers=headers)
    try:
        return json.loads(response.text)
    except Exception as e:
        print(e)
        print("response: %s" % response.text)
        traceback.print_exc()


def __get_db_dict():
    """
    Get the list of sources as a json, e.g:
    
    {
        "monit_idb_monitoring": {
            "type": "influxdb", 
            "id": 9109, 
            "database": "monit_production_monitoring"
        }, 
        "monit_idb_cmsjm_old": {
            "type": "influxdb", 
            "id": 8991, 
            "database": "monit_production_condor"
        }
    }
    """
    _default_path = os.path.join(
        os.path.dirname(os.path.abspath(__file__)), "../../../static/datasources.json"
    )
    _path = os.getenv("MONIT_DB_DICT_FILE", _default_path)
    try:
        with open(_path, "r") as dir_file:
            return json.load(dir_file)
    except (ValueError, OSError):
        print("Failed to load the datasources directory")
        return None


def __infer_index(_db, dbname):
    _name_str = (
        re.match(r"\[{0,1}([^\]]*)\]{0,1}", _db.get("database")).group(1)
        if _db
        else dbname
    )
    values = [index + "*" if "*" not in index else index for index in _name_str.split(",")]
    return values


def main():
    """Main function"""
    optmgr = OptionParser()
    opts = optmgr.parser.parse_args()
    if opts.list_dbs:
        print(json.dumps(__get_db_dict(), indent=2))
        sys.exit(0)
    dbid = int(opts.dbid)
    idx = int(opts.idx)
    limit = int(opts.limit)
    verbose = int(opts.verbose)
    dbname = opts.dbname
    query = opts.query
    token = (
        open(opts.token).read().replace("\n", "")
        if os.path.exists(opts.token)
        else opts.token
    )
    if dbname == "cms":
        dbname = "es-cms"  # Now we use the actual name in the dictionary.
    results = run(opts.url, token, dbid, dbname, query, idx, limit, verbose)
    print(json.dumps(results))


if __name__ == "__main__":
    main()
