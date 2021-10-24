#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""
This script compares two ES instances
index by index
requires elasticsearch, click
"""
import sys
import json
from datetime import datetime, timezone, timedelta
import elasticsearch
import elasticsearch.helpers
import click

_VALID_DATE_FORMATS = ["%Y/%m/%d", "%Y-%m-%d", "%Y%m%d"]


def compare_es(
    es1_url,
    es2_url,
    start_date,
    end_date,
    index_prefix="cms",
    verbose=False,
    credentials_path=".secrets/escms.json",
):
    """
    Compare two ES instances, index by index, between two dates. 
    Args:
        - es1_url url of the original ES
        - es2_url url of the new ES
        - start_date datetime object representing the start of the date range
        - end_date datetime object representing the end of the date range
        - index_prefix prefix of the indexes to compare
        - verbose print additional info, as the document ids
        - credentials_path path to a file containing a json array
                            ["username", "password"]
                       
    """
    es1_client = get_client(es1_url, credentials_path)
    es2_client = get_client(es2_url, credentials_path)
    delta = timedelta(days=1)
    days = (end_date - start_date).days + 1
    print("index", "# missing", "ids_es1", "ids_es2")
    for day in (start_date + timedelta(days=n) for n in range(days)):
        _index = f"{index_prefix}-{day.strftime('%Y-%m-%d')}"
        ids_es1 = query_es(es1_client, _index)
        ids_es2 = query_es(es2_client, _index)
        _only_1 = ids_es1 - ids_es2
        _only_2 = ids_es2 - ids_es1
        missing = _only_1.union(_only_2)
        if missing:
            print(_index, len(missing), len(ids_es1), len(ids_es2))
            if verbose:
                print("Only {es1_url}", _only_1, file=sys.stderr)
                print("Only {es2_url}", _only_2, file=sys.stderr)


def get_client(url, credentials_path):
    """
    get a elasticsearch client for the given url
    """
    es_credentials = ()
    es_client = None
    with open(credentials_path, "r") as f:
        es_credentials = tuple(json.load(f))
        es_client = elasticsearch.client.Elasticsearch(
            [url],
            use_ssl=True,
            http_auth=es_credentials,
            ca_certs="/etc/pki/ca-trust/extracted/openssl/ca-bundle.trust.crt",
        )
    return es_client


def query_es(es_client, index):
    """
    Query an specific index in a given ES instance.
    """
    _query = {
        "query": {
            "bool": {
                "filter": [
                    {
                        "query_string": {
                            "analyze_wildcard": True,
                            "query": f'_index:"{index}"',
                        }
                    }
                ]
            }
        },
        "_source": ["GlobalJobId"],
    }
    scan = elasticsearch.helpers.scan(
        es_client, index=index, query=_query, timeout="10m", request_timeout=600
    )
    return set([d["_id"] for d in scan])


@click.command()
@click.argument("start_date", nargs=1, type=click.DateTime(_VALID_DATE_FORMATS))
@click.argument("end_date", nargs=1, type=click.DateTime(_VALID_DATE_FORMATS))
@click.option("--es1", default="es-cms.cern.ch:9203", help="URL of the original ES")
@click.option("--es2", default="es-cms7.cern.ch:9203", help="URL of the new ES")
@click.option("--index_prefix", default="cms", help="prefix of the indexes")
@click.option(
    "--credentials_path",
    default=".secrets/escms.json",
    help="""Path for the credentials file, 
    the credentials file contains the pair ["username", "password"]
    """,
)
@click.option(
    "--verbose",
    default=False,
    is_flag=True,
    help="Print the ids of the missing documents",
)
def main(start_date, end_date, es1, es2, index_prefix, verbose, credentials_path):
    compare_es(
        es1,
        es2,
        start_date,
        end_date,
        index_prefix=index_prefix,
        verbose=verbose,
        credentials_path=credentials_path,
    )


if __name__ == "__main__":
    main()
