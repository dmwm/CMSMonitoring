#!/usr/bin/env python
# -*- coding: utf-8 -*-
import os
import sys
import argparse
import traceback
import logging
from grafanamanager import GrafanaManager

logging.basicConfig(level=os.environ.get("LOGLEVEL", "DEBUG"))


class OptionParser:
    def __init__(self):
        """User based option parser"""
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
            help="if --uid is specified, this will be the title of the new dashboard, "
                 "if --query is used this value will be used as postfix",
        )
        self.parser.add_argument(
            "--datasource_replacement",
            action="append",
            nargs=2,
            dest="replacements",
            default=None,
            help="pairs of values, 'orig' 'new_ds', to replace a datasource in the dashboard(s). "
                 "you can use several values using --datasource_replacement 'cmsweb-k8s' 'cmsweb-k8s-new' "
                 "--datasource_replacement 'monit_es_condor_2019' 'monit_es_condor'",
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


def main():
    """
    Main function
    """
    optmgr = OptionParser()
    opts = optmgr.parser.parse_args()
    if not opts.token:
        print("A grafana token is required (either using the "
              "--grafana_token option or seting the GRAFANA_TOKEN environment variable")
        sys.exit(1)
    if not (opts.dashboard_uid or opts.dashboards_query):
        print("You need to set either --uid or --query")
        sys.exit(1)
    try:
        # print(opts)
        mgr = GrafanaManager(
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
