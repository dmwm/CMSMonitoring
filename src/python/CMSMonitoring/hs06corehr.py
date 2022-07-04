#!/usr/bin/env python
# coding: utf-8

"""
File:           hs06corehr.py
Author:         Efe Yazgan <efe.yazgan AT cern dot ch>,
                Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>
Description: This script generates sum of HS06CoreHr metric for each Campaign and its 
 WMAgent_SubTaskName(s) in last 2 years. Max bucket size in ElasticSearch is 
"""

import json
import os
import requests

# Base lucene query for filtering  ElasticSearch data.
BASE_QUERY = 'Type:production AND Status:Completed AND Campaign:\\"RunIISummer19UL17wmLHEGEN\\"'
# BASE_QUERY = 'Type:production AND Status:Completed AND
#    Campaign:(\\"RunIISummer19UL17wmLHEGEN\\" OR \\"RunIISummer19UL17wmLHEGEN\\")'

# Query is used to get all WMAgent_SubTaskName values
query_for_wmagent_subtasknames = {
    "size": 10000,
    "query": {
        "bool": {
            "filter": [
                {
                    "range": {
                        "RecordTime": {
                            "gt": "ITERATION_INTERVAL_GT",
                            "lte": "ITERATION_INTERVAL_LTE",
                            "format": "epoch_millis"
                        }
                    }
                },
                {
                    "query_string": {
                        "analyze_wildcard": True,
                        "query": "VAR_BASE_QUERY"
                    }
                }
            ]
        }
    },
    "aggs": {
        "agg_sub_task_name": {
            "terms": {
                "field": "WMAgent_SubTaskName",
                "size": 10000,
                "order": {
                    "_key": "asc"
                },
                "min_doc_count": 1
            }
        }
    }
}

# Query is used to get all aggregation results of sum HS06CoreHr,
# grouped by Campaign and WMAgent_SubTaskName.
main_query = {
    "size": 10000,
    "query": {
        "bool": {
            "filter": [
                {
                    "range": {
                        "RecordTime": {
                            "gt": "now-2y",
                            "lte": "now",
                            "format": "epoch_millis"
                        }
                    }
                },
                {
                    "query_string": {
                        "analyze_wildcard": True,
                        "query": "WMAgent_SubTaskName:(VAR_WMAgent_SubTaskName) AND VAR_BASE_QUERY"
                    }
                }
            ]
        }
    },
    "aggs": {
        "agg_campaign": {
            "terms": {
                "field": "Campaign",
                "size": 1000000,
                "order": {
                    "_key": "asc"
                },
                "min_doc_count": 1
            },
            "aggs": {
                "agg_sub_task_name": {
                    "terms": {
                        "field": "WMAgent_SubTaskName",
                        "size": 1000000,
                        "order": {
                            "_key": "asc"
                        },
                        "min_doc_count": 1
                    },
                    "aggs": {
                        "agg_sum_of_HS06CoreHr": {
                            "sum": {
                                "field": "HS06CoreHr"
                            }
                        }
                    }
                }
            }
        }
    }
}

# Number of WMAgent_SubTaskName in each iteration.
BATCH_SIZE = 1000
url = "https://monit-grafana.cern.ch/api/datasources/proxy/8983/_msearch"
payload_index_props = {"search_type": "query_then_fetch", "ignore_unavailable": True, "index": ["cms-20*"]}
headers = {
    "Content-Type": "application/json",
    "Authorization": "Bearer {}".format(os.environ["GRAFANA_VIEWER_TOKEN"])
}


def chunks(lst, n):
    """Yield successive n-sized chunks from l."""
    for i in range(0, len(lst), n):
        yield lst[i:i + n]


def wmagent_subtasknames(start=0, stop=(365 + 365), step=30, time_marker_str="d"):
    """Prepare list of queries to fetch all wmagent_subtasknames in batches."""
    payload_wmagent_subtasknames = json.dumps(payload_index_props) + " \n" + json.dumps(
        query_for_wmagent_subtasknames) + "\n"
    payload_wmagent_subtasknames = payload_wmagent_subtasknames.replace("VAR_BASE_QUERY", BASE_QUERY)
    tquery = "now-{n}" + time_marker_str  # i.e now-30d
    payloads_and_ranges = []
    for n in range(start, stop, step):
        # i.e: (now-0d, now-30d)
        range_tuple = (tquery.format(n=n), tquery.format(n=min(n + step, stop)))
        tpayload = payload_wmagent_subtasknames.replace("ITERATION_INTERVAL_GT", range_tuple[1]).replace(
            "ITERATION_INTERVAL_LTE", range_tuple[0])
        payloads_and_ranges.append((tpayload, range_tuple))
    return payloads_and_ranges


def iterate_payloads_of_wmagent_subtasknames(payloads_of_wmagent_subtasknames):
    """All wmagent_subtasknames will be fetched."""
    all_buckets = []
    try:
        for pyld, trange in payloads_of_wmagent_subtasknames:
            response = requests.request("POST", url, headers=headers, data=pyld)
            result = response.json()
            tmp_buckets = result["responses"][0]["aggregations"]["agg_sub_task_name"]["buckets"]
            print(trange, "=> bucket size:", len(tmp_buckets))
            all_buckets.extend(tmp_buckets)
        return all_buckets
    except IndexError:
        print("All time ranges are queried.")
        return all_buckets


def main_queries(wmagent_subtasknames_list):
    """ES does not support more than 10k aggregation result. Hence, we will get
    all WMAgent_SubTaskName values in iterative way. Because our aggregation is
    the group by of WMAgent_SubTaskName, response bucket size will not be greater
    than our BATCH_SIZE.
    """
    main_payload = json.dumps(payload_index_props) + " \n" + json.dumps(main_query) + "\n"
    main_payload = main_payload.replace("VAR_BASE_QUERY", BASE_QUERY)
    payloads = []
    for batch in chunks(wmagent_subtasknames_list, BATCH_SIZE):
        # i.e: (now-0d, now-30d)
        tpayload = main_payload.replace("VAR_WMAgent_SubTaskName", '\\"' + '\\" OR \\"'.join(batch) + '\\"').replace(
            "VAR_BASE_QUERY", BASE_QUERY)
        payloads.append(tpayload)
    return payloads


def iterate_queries(main_payloads):
    all_buckets = []
    try:
        for pyld in main_payloads:
            response = requests.request("POST", url, headers=headers, data=pyld)
            result = response.json()
            tmp_buckets = result["responses"][0]["aggregations"]["agg_campaign"]["buckets"][0]["agg_sub_task_name"][
                "buckets"]
            print("=> bucket size:", len(tmp_buckets))
            all_buckets.extend(tmp_buckets)
        return all_buckets
    except IndexError:
        print("All time ranges are queried.")
        return all_buckets


def presenter(buckets):
    list_prepid_hs06 = []
    for sub_task in buckets:
        # print("Campaign: " + campaign["key"] + " SubTask: " + sub_task["key"], " ")
        # print("SubTask: ",str(sub_task["key"]).split("/")[2],
        # " HS06CoreHr", sub_task["agg_sum_of_HS06CoreHr"]["value"])
        newtuple = str(sub_task["key"]).split("/")[2] + " = " + str(sub_task["agg_sum_of_HS06CoreHr"]["value"])
        list_prepid_hs06.append(newtuple)
    grouped = {}
    for x in list_prepid_hs06:
        key = x.partition("_")[0]
        grouped.setdefault(key, []).append(x)
    grouped = grouped.values()
    for x in grouped:
        total_hs06 = 0
        for y in x:
            listtmp = y.split(" = ")
            total_hs06 += float(listtmp[1])
            print(y)
        print("total Hs06=", total_hs06)
        print("-" * 79)


def main():
    # Make unique list of wmagent_subtasknames
    wmagent_subtasknames_list = list(
        set([i["key"] for i in iterate_payloads_of_wmagent_subtasknames(wmagent_subtasknames())]))
    queries = main_queries(wmagent_subtasknames_list)
    all_buckets = iterate_queries(queries)
    print("=" * 32, " Presentation: ", "=" * 32, "\n", "=" * 79, end="\n", sep="")
    presenter(all_buckets)
    print("Total aggregation result size: ", len(all_buckets))


if __name__ == "__main__":
    main()
