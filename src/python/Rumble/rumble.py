#!/usr/bin/env python3
"""
Author:      Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>
Description: Client to run jsoniq queries in rumble server.
"""

# system modules
import os
import json
import argparse

# third party libs
import requests


class OptionParser:
    def __init__(self):
        """ User based option parser """
        # Max output size is unlimited
        default_url = "https://cms-monitoring.cern.ch/"
        uri = default_url + "jsoniq?overwrite=yes&materialization-cap=-1"
        self.parser = argparse.ArgumentParser(prog='PROG')
        self.parser.add_argument("--query", action="store", dest="query", default="", help="Input: file or query")
        self.parser.add_argument("--output", action="store", dest="output", default=None, help="Output file")
        self.parser.add_argument("--server", action="store", dest="server", default=uri, help="Rumble server")


def handle_output_file(output_file):
    """ Handles output file """
    # Accepts both given full path of file or
    # only file name which will be created in current directory.
    if os.path.isdir(os.path.abspath(os.path.join(output_file, os.pardir))):
        return output_file
    else:
        print("Given output file path is not exist or not reachable:", output_file)
        print("Writing to default path: ", os.path.join(os.getcwd(), "rumble_output.txt"))
        return os.path.join(os.getcwd(), "rumble_output.txt")


def rumble(server, query, output=None):
    response = json.loads(requests.post(server, data=query).text)
    if 'warning' in response:
        print(json.dumps(response['warning']))
    if 'values' in response:
        if output:
            output_file = handle_output_file(output)
            try:
                with open(output_file, "w+") as f:
                    f.write(json.dumps(response))
                    return "Successfully written to: " + str(os.path.join(os.getcwd(), output))
            except Exception as e:
                print("Could not write to file:", output_file)
                print(e)
        else:
            for e in response['values']:
                return json.dumps(e)
    elif 'error-message' in response:
        return response['error-message']
    else:
        return response


def main():
    """ Main function """
    optmgr = OptionParser()
    opts = optmgr.parser.parse_args()
    if os.path.exists(opts.query):  # Get query from file
        server = opts.server
        query = open(opts.query).read()
        print("<Input Query>:\n" + str(query))
    elif opts.query.startswith("hdfs://"):
        server = opts.server + "&query-path=" + opts.query
        query = None
        print("<Input Query in HDFS>:", opts.query)
    else:
        server = opts.server
        query = opts.query
        print("<Input Query>:\n" + str(query))
    # Get output path as output
    if opts.output:
        output = opts.output
    else:
        print("Output path is not given. Result will be stdout!")
        output = None
    resp = rumble(server, query, output)
    if resp:
        print("<Rumble Response Values>")
        print(resp)


if __name__ == '__main__':
    main()

