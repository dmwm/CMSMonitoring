#!/usr/bin/env python
# -*- coding: utf-8 -*-
# Author: Christian Ariza <christian.ariza AT gmail [DOT] com>
import os
import sys
import json
import argparse
import requests


class OptionParser:
    def __init__(self):
        """User based option parser"""
        desc = """
This app creates a json file with the name, id, and type of datasource in the user organization. 
It requires a admin token from grafana. The token can be set either using the --token option 
or through the GRAFANA_ADMIN_TOKEN environment variable. 
               """
        self.parser = argparse.ArgumentParser(prog="PROG", usage=desc)
        self.parser.add_argument(
            "--token", action="store", dest="token", default=None,
            help="Admin token: either file with token or token string"
        )
        self.parser.add_argument(
            "--url",
            action="store",
            dest="url",
            default="https://monit-grafana.cern.ch",
            help="MONIT URL",
        )
        self.parser.add_argument(
            "--output", action="store", dest="output", default=None, help="output file"
        )


def get_datasources(token, base="https://monit-grafana.cern.ch"):
    headers = {
        "Authorization": "Bearer {}".format(token),
        "Content-type": "application/x-ndjson",
        "Accept": "application/json",
    }
    uri = base + "/api/datasources"
    response = requests.get(uri, headers=headers)
    try:
        fullResponse = json.loads(response.text)
    except Exception as e:
        print("Fail to load HTTP response as JSON")
        print(response.text)
        sys.exit(1)
    return {
        x["name"]: {"id": x["id"], "type": x["type"], "database": x["database"]}
        for x in fullResponse
    }


def main():
    """Main function"""
    optmgr = OptionParser()
    opts = optmgr.parser.parse_args()
    token = os.getenv("GRAFANA_ADMIN_TOKEN", opts.token)
    if os.path.exists(token):  # if token is a file
        token = open(token).readline().replace('\n', '')
    output = opts.output
    base = opts.url
    if not token:
        print(
            "The token is required either using --token with the value or using the GRAFANA_ADMIN_TOKEN env variable"
        )
        sys.exit(-1)
    datasources = get_datasources(token, base=base)
    if not output:
        json.dump(datasources, sys.stdout, indent=4)
    else:
        with open(output, "w") as _output_file:
            json.dump(datasources, _output_file, indent=4)


if __name__ == "__main__":
    main()
