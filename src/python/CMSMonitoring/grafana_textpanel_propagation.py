#!/usr/bin/env python
# -*- coding: utf-8 -*-
import os
import sys
import json
import argparse
import traceback
import logging
from grafanamanager import GrafanaManager

logging.basicConfig(level=os.environ.get("LOGLEVEL", "DEBUG"))


class OptionParser:
    def __init__(self):
        """User based option parser"""
        desc = """
This app allows update text on panels from several dashboards
based on a template, given than the text panels have a fixed start_text/end_text.
               """
        self.parser = argparse.ArgumentParser(prog="grafana text propagation", usage=desc)
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
            "--source_query",
            action="store",
            dest="source_query",
            default='{"query":"HomeTabs"}',
            help="""query string for the source dashboard, it must have json format, 
            and contain at least one of the following keys:
            - query
            - tags
            - uid
            for example '{"query":"HomeTabs"}'
            """,
        )
        self.parser.add_argument(
            "--target_query",
            action="store",
            dest="target_query",
            default='{"tags":"home"}',
            help="""query string for the target dashboards, it must have json format, 
            and contain at least one of the following keys:
            - query
            - tags
            - uid
            for example '{"tags":"home"}'
            """,
        )
        self.parser.add_argument(
            "--start_text",
            action="store",
            help="String that marks the start of the text to replace, default <!--START_MENU-->",
            default="<!--START_MENU-->",
        )
        self.parser.add_argument(
            "--end_text",
            action="store",
            help="String that marks the end of the text to replace, default <!--END_MENU-->",
            default="<!--END_MENU-->",
        )


def main():
    """
    Main function
    """
    optmgr = OptionParser()
    opts = optmgr.parser.parse_args()
    if not opts.token:
        print(
            "A grafana token is required (either using the --grafana_token option "
            "or setting the GRAFANA_TOKEN environment variable"
        )
        sys.exit(1)
    try:
        print(opts)
        mgr = GrafanaManager(
            grafana_url=opts.url, grafana_token=opts.token
        )
        updated_db = mgr.text_template_replace(
            json.loads(opts.source_query),
            json.loads(opts.target_query),
            start_text=opts.start_text,
            end_text=opts.end_text)
        print(f"{updated_db} dashboards {'were' if updated_db > 1 else 'was'} updated")
    except Exception as e:
        print(e)
        traceback.print_exc()


if __name__ == "__main__":
    main()
