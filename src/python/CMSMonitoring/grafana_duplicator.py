#!/usr/bin/env python
# -*- coding: utf-8 -*-
import os
import sys
import requests
import json
import argparse
import traceback
import logging
logging.basicConfig(level=os.environ.get("LOGLEVEL", "DEBUG"))

class OptionParser:
    def __init__(self):
        "User based option parser"
        desc = """
This app allows to create a copy of one (using the uid parameter) 
or a set of dashboards (using the query parameter) 
allowing to change the datasource and the id and the title. 
               """
        self.parser = argparse.ArgumentParser(prog="grafana duplicator", usage=desc)
        self.parser.add_argument(
            "--token",
            action="store",
            dest="token",
            default=os.getenv("GRAFANA_TOKEN", None),
            help="Admin token",
        )
        self.parser.add_argument(
            "--url",
            action="store",
            dest="url",
            default="https://monit-grafana-dev.cern.ch",
            help="MONIT URL",
        )
        self.parser.add_argument(
            "--output",
            action="store",
            dest="output",
            default="./output",
            help="output folder for the local copy of the dashboards",
        )
        self.parser.add_argument(
            "--uid",
            action="store",
            dest="dashboard_uid",
            default=None,
            help="uid of the dashboard",
        )
        self.parser.add_argument(
            "--title",
            action="store",
            dest="new_title",
            default=None,
            help="if --uid is specified, this will be the title of the new dashboard, if --query is used this value will be used as postfix",
        )
        self.parser.add_argument(
            "--datasource_replacement",
            action="append",
            nargs=2,
            dest="replacements",
            default=None,
            help="""pairs of values, 'orig' 'new_ds', to replace a datasource in the dashboard(s).
you can use several values using --datasource_replacement "cmsweb-k8s" "cmsweb-k8s-new" --datasource_replacement "monit_es_condor_2019" "monit_es_condor" 
""",
        )
        self.parser.add_argument(
            "--query",
            action="store",
            dest="dashboards_query",
            default=None,
            help="query string for the dashboards",
        )
        self.parser.add_argument(
            "--store_only",
            action="store_true",
            help="Only backup the matching dashboards with the specified changes, if any",
            default=False,
        )


class grafana_manager:
    def __init__(self, grafana_url=None, grafana_token=None, output_folder="./output"):
        assert (
            grafana_token and grafana_url
        ), "both grafana_token and grafana_url are required."
        self.GRAFANA_TOKEN = grafana_token
        self.GRAFANA_URL = (
            grafana_url[:-1] if grafana_url.endswith("/") else grafana_url
        )
        self.OUTPUT_FOLDER = output_folder

    def dashboard_duplicator(
        self,
        dashboard_uid=None,
        dashboards_query=None,
        datasources_replacements=None,
        title=None,
        store_only=False,
    ):
        assert (
            dashboard_uid or dashboards_query
        ), "Either uid or query need to be specified"
        replace_title = False
        append_title = False
        if title:
            replace_title = True if dashboard_uid else False
            append_title = not replace_title
        dashboards = []
        if dashboards_query:
            dashboards = self.search_dasboards(
                query=dashboards_query, uid=dashboard_uid
            )
        if dashboard_uid:
            dashboards.append(self.get_dashboard(dashboard_uid))
        # print(json.dumps(dashboards, indent=4))
        replacements = (
            dict(datasources_replacements) if datasources_replacements else {}
        )
        logging.debug('replacements {}', replacements)
        for d in dashboards:
            # Save a backup
            self.__save("backup", d)
            # Create the new dashboard
            new_dashboard = {
                "dashboard": d["dashboard"],
                "overwrite": False,
                "folderId": d["meta"]["folderId"],
            }
            new_dashboard["dashboard"]["id"] = None
            new_dashboard["dashboard"]["uid"] = None
            new_dashboard
            new_title = (
                title
                if replace_title
                else "{}_{}".format(new_dashboard["dashboard"]["title"], title)
                if append_title
                else new_dashboard["dashboard"]["title"]
            )
            new_dashboard["dashboard"]["title"] = new_title
            self.__replace_dataset(new_dashboard, replacements)
            self.__save("new", new_dashboard)
            if not store_only:
                logging.info("Try to create the dashboards in grafana")
                logging.info(new_title)
                logging.info(self.post_dashboard(new_dashboard))

    def search_dasboards(self, query=None, uid=None):
        params = {}
        request_uri = None
        request_uri = "{uri}/api/search".format(uri=self.GRAFANA_URL)
        params["type"] = "dash_db"
        params["query"] = query
        dashboards_links = self.__make_request(request_uri, params=params)
        return [self.get_dashboard(x["uid"]) for x in dashboards_links]

    def get_dashboard(self, uid=None):
        request_uri = "{uri}/api/dashboards/uid/{uid}".format(
            uri=self.GRAFANA_URL, uid=uid
        )
        return self.__make_request(request_uri, params=None)

    def post_dashboard(self, dashboard):
        request_uri = "{uri}/api/dashboards/db".format(uri=self.GRAFANA_URL)
        return self.__make_request(request_uri, params=dashboard, type="post")

    def __make_request(self, uri, params, type="get"):
        headers = {"Authorization": "Bearer {}".format(self.GRAFANA_TOKEN)}
        verify = True
        # The monit-grafana-dev uses an invalid certificate
        if "-dev" in uri:
            verify = False
        response = None
        if type == "get":
            response = requests.get(uri, params=params, headers=headers, verify=verify)
        elif type == "post":
            response = requests.post(uri, json=params, headers=headers, verify=verify)
        if response:
            return json.loads(response.content)
        else:
            raise Exception("Something happen with the request, {}".format(response))

    def __save(self, subfolder, d):
        _folderId = (
            d["meta"]["folderId"] if "meta" in d else d.get("folderId", "unknown")
        )
        filename = os.path.join(
            self.OUTPUT_FOLDER,
            subfolder,
            "{folderId}_{title}.json".format(
                folderId=_folderId, title=d["dashboard"].get("title", "unknown")
            ),
        )
        os.makedirs(os.path.dirname(filename), exist_ok=True)
        with open(filename, "w") as f:
            json.dump(d, f, indent=4)

    def __replace_dataset(self, element, replacements):
        if isinstance(element, list):
            for e in element:
                self.__replace_dataset(e, replacements)
        elif isinstance(element, dict):
            for key in element:
                if key == "datasource":
                    element[key] = replacements.get(element[key], element[key])
                elif isinstance(element[key], (dict, list)):
                    self.__replace_dataset(element[key], replacements)

def main():
    """
    Main function
    """
    optmgr = OptionParser()
    opts = optmgr.parser.parse_args()
    if not opts.token:
        print(
            "A grafana token is required (either using the --grafana_token option or seting the GRAFANA_TOKEN environment variable"
        )
        sys.exit(1)
    if not (opts.dashboard_uid or opts.dashboards_query):
        print("You need to set either --uid or --query")
        sys.exit(1)
    try:
        # print(opts)
        mgr = grafana_manager(
            grafana_url=opts.url, grafana_token=opts.token, output_folder=opts.output
        )
        mgr.dashboard_duplicator(
            dashboard_uid=opts.dashboard_uid,
            dashboards_query=opts.dashboards_query,
            datasources_replacements=opts.replacements,
            title=opts.new_title,
            store_only=opts.store_only,
        )
    except Exception as e:
        print(e)
        traceback.print_exc()


if __name__ == "__main__":
    main()
