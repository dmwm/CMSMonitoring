# !/usr/bin/env python
# -*- coding: utf-8 -*-
# Author: Ceyhun Uzunoglu <ceyhunuzngl AT gmail [DOT] com>
# Create html table for EOS paths' size as a result of xrdcp command
#
# acronjob:
#       - $HOME/CMSMonitoring/scripts/eos_path_size.sh
# How it works:
#       - Gets only paths from XRDCP command
#       - All used values are results of EOS command
#       - This script can use direct results of `eos` command or a file that includes the result of `eos` command
# Parameters:
#       - --output_file: Sets the output html
#       - --input_eos_file: If provided, the file that contains EOS command results will be used.
#                         This file is updated every 10 minutes by VOCMS team and they are managing the cron job
#                         If not provided, make sure that you are a quota admin to run `eos` command


import json
import os
import sys
from datetime import datetime

import click
import numpy as np
import pandas as pd
from schema import Schema, Use, SchemaError

pd.options.display.float_format = "{:,.2f}".format
pd.set_option("display.max_colwidth", -1)

EXCLUDED_PATHS = ["/eos/cms/store/cmst3", "/eos/recovered", "/eos/totem"]
RECYCLE = "/eos/cms/proc/recycle/"
TB_DENOMINATOR = 10 ** 12


def tstamp():
    """Return timestamp for logging"""
    return datetime.utcnow().strftime('%Y-%m-%dT%H:%M:%SZ')


def get_update_time(input_eos_file):
    """Create update time depending on reading EOS results from file or directly from command"""
    if input_eos_file:
        # Set update time to eos file modification time if file input used
        try:
            return datetime.utcfromtimestamp(os.path.getmtime(input_eos_file)).strftime('%Y-%m-%dT%H:%M:%SZ')
        except OSError as e:
            print(tstamp(), "ERROR: coul not get last modification time of file:", str(e))
    else:
        return datetime.utcnow().strftime("%Y-%m-%dT%H:%M:%SZ")


# EOS operation
def get_validated_eos_results(eos_lines_list):
    """Validate, convert types and filter

    Example line: "quota=node uid=akhukhun space=/eos/cms/store/ usedbytes=0 ..."
    Filter: "gid=ALL" or "gid=project" filter is applied
    """

    schema = Schema(
        [
            {
                'quota': str,
                'gid': Use(str),
                'space': Use(str),
                'usedbytes': Use(float),
                'usedlogicalbytes': Use(float),
                'usedfiles': Use(float),
                'maxbytes': Use(float),
                'maxlogicalbytes': Use(float),
                'maxfiles': Use(float),
                'percentageusedbytes': Use(float),
                'statusbytes': str,
                'statusfiles': str,
                # No 'uid'
            }
        ]
    )

    # Get only rows that contain "gid" and it's value should be "ALL" or "project"
    dict_array = list(
        map(
            lambda line: dict(tuple(s.split("=")) for s in line.strip().split(' ')),
            [row for row in eos_lines_list if (("gid=ALL" in row) or ("gid=project" in row))]
        )
    )

    try:
        # Validate and convert types
        return schema.validate(dict_array)
    except SchemaError as e:
        print(tstamp(), "Data is not valid:", str(e))
        sys.exit(1)


def get_eos(file_path=None):
    """Get EOS output raw results from either eos command or from a file that contain those results"""
    if file_path:
        print(tstamp(), "EOS file path is provided, reading from file")
        # Read eos result from file
        with open(file_path) as file:
            return get_validated_eos_results(file.readlines())
    else:
        print(tstamp(), "EOS file path is NOT provided")
        print(tstamp(), "Running eos quota ls command")
        # For VOC team: run eos command and get output
        try:
            # export EOSHOME="" is needed to avoid warning messages
            return get_validated_eos_results(
                str(os.system('export EOSHOME="" && eos -r 103074 1399 quota ls -m')).split("\n"))
        except OSError as e:
            print(tstamp(), 'ERROR: Cannot get the eos quota ls output from EOS:', str(e))
            sys.exit(1)


def create_eos_df(file_path):
    """Create dataframe from EOS output lines"""
    df = pd.DataFrame(get_eos(file_path)).rename(columns={'space': 'path'})
    # Re-order and drop unwanted columns: 'usedfiles', 'statusbytes', 'statusfiles', 'quota', 'gid'
    return df[['path', 'usedlogicalbytes', 'maxlogicalbytes', 'usedbytes', 'maxbytes', 'percentageusedbytes']]


def get_paths_from_xrdcp(json_data):
    """Get paths from XRDCP command and filter our cmst3 paths """
    data = json.load(json_data)
    paths = [elem["path"][0] for elem in data['storageservice']['storageshares']]
    # Filter out cmst3/*
    paths = [path for path in paths if all((exc not in path) for exc in EXCLUDED_PATHS)]
    return paths


def create_html(df, update_time, total_row):
    """Create html page with given dataframe

    Notes :
        CSS
            `style="width:60%`: 60% is pretty
            `dataTables_filter input`: search bar settings
            `td:nth-child(n+2)`: align numbers to right
            `white-space: nowrap`: do not carriage return, not break line
    """

    total_on_header = """<tr >
      <th>{Path}</th>
      <th>{logical used(TB)}</th>
      <th>{logical quota(TB)}</th>
      <th>{raw used(TB)}</th>
      <th>{raw quota(TB)}</th>
      <th>{used/quota}</th>
      <th>{logical/raw (used)}</th>
      <th>{logical/raw (quotas)}</th>
    </tr>
 </thead>
    """.format(**total_row)

    main_column = df["Path"].copy()
    df["Path"] = (
        '<a class="Path">'
        + main_column
        + '</a><br>'
    )

    html = df.to_html(escape=False, index=False)

    # Add total ro to header. Be careful there should be only one  <thead>...</thead>!
    html = html.replace(" </thead>", total_on_header)

    html = html.replace(
        'table border="1" class="dataframe"',
        'table id="dataframe" class="display compact" style="width:60%;"',
    )
    html = html.replace('style="text-align: right;"', "")
    # cleanup of the default dump
    html = html.replace(
        'table border="1" class="dataframe"',
        'table id="dataframe" class="display compact" style="width:100%;"',
    )
    html = html.replace('style="text-align: right;"', "")

    html_header = f"""<!DOCTYPE html>
        <html>
        <head>
        <link rel="stylesheet" href="https://www.w3schools.com/w3css/4/w3.css">
        <meta name="viewport" content="width=device-width, initial-scale=1">
        <link rel="stylesheet" href="https://cdn.datatables.net/1.10.20/css/jquery.dataTables.min.css">
        <style>
            .dataTables_filter input {{
              border: 7px solid Tomato;
              width: 400px;
              font-size: 16px;
              font-weight: bold;
            }}
            table td {{
            word-break: break-all;
            }}
            table td:nth-child(n+2) {{
                text-align: right;
            }}
            table td:nth-child(1) {{
                font-weight: bold;;
            }}
            #dataframe tr:nth-child(even) {{
              background-color: #dddfff;
            }}
            #dataframe tr td {{
              width: 1%;
              white-space: nowrap;
            }}
            small {{
                font-size : 0.4em;
            }}
        </style>
        </head>
        <body>
            <div class="cms">
                <img src="https://cds.cern.ch/record/1306150/files/cmsLogo_image.jpg"
                    alt="CMS" style="width: 5%; float:left">
                <h1 style="width: 95%; float:right">
                    EOSCMS quotas and usage monitoring
                    <small>Last Update: {update_time} UTC</small>
                </h1>
            </div>
            <div class="w3-container">
            <ul class="w3-ul w3-small" style="width:70%;margin-left:0%">
              <li class="w3-padding-small">
                This page shows the size of CMS EOS paths in TB. Bytes to TB denominator is <b>10^12</b>.
              </li>
              <li class="w3-padding-small">
                All values come from: <b>eos -r 103074 1399 quota ls -m</b> and paths come from:
                <b>xrdcp -s root://eoscms.cern.ch//eos/cms/proc/accounting -</b> 
                 </pre>
              </li>
              <li class="w3-padding-small">
                [NOTE-1]:<b>{"*,".join(EXCLUDED_PATHS) + "*"}</b> paths are filtered out and not used in calculations.
              </li>
              <li class="w3-padding-small">
                [NOTE-2]: The sizes of "/eos/cms/proc/recycle/" are divided by <strong>2</strong> for obvious reasons.
              </li>
              <li class="w3-padding-small">
                Script source code: <b>
                <a href="https://github.com/dmwm/CMSMonitoring/blob/master/src/python/CMSMonitoring/eos_path_size.py">
                        eos_path_size.py</a></b>
              </li>
            </ul>
            </div>
         """
    html_middle = (
        '''
        <div class="container" style="display:block; width:100%">
    ''')
    html_footer = (
        '''
        </div>
        <script src="https://ajax.googleapis.com/ajax/libs/jquery/3.3.1/jquery.min.js"></script>
        <script type="text/javascript" src="https://cdn.datatables.net/1.10.20/js/jquery.dataTables.min.js"></script>
        <script>
            $(document).ready(function () {
                var dt = $('#dataframe').DataTable( {
                    "orderCellsTop": true,
                    "dom": "lifrtip",
                    "order": [[ 1, "desc" ]],
                    "pageLength" : 300,
                    "scrollX": false,
                    language: {
                        search: "_INPUT_",
                        searchPlaceholder: "--- Search EOS Path ---",
                    },
                });
            });
        </script>
    </body>
    <!-- final -->
    </html>'''
    )

    result = html_header + html_middle + html + html_footer
    return result


@click.command()
@click.option("--output_file", default=None, required=True, help="For example: /eos/.../www/test/test.html")
@click.option("--input_eos_file",
              default=None,
              required=False,
              help="Result of 'eos -r 103074 1399 quota ls -m', i.e.: /eos/cms/store/accounting/eos_quota_ls.txt")
def main(output_file=None, input_eos_file=None):
    """
        Main function combines xrdcp and EOS results then creates HTML page
    """

    # Get xrdcp command output as input to python script.
    xrdcp_paths = get_paths_from_xrdcp(sys.stdin)

    # Get EOS values as pandas dataframe either from file or directly from EOS command
    df = create_eos_df(input_eos_file)

    # Filter dataframe to only include paths from xrdcp
    df = df[df['path'].isin(xrdcp_paths)]

    #  RECYCLE: divide these columns to 2
    cols_to_divide = ["usedlogicalbytes", "maxlogicalbytes", "usedbytes", "maxbytes"]

    # Resetting index sets index number to 0, so get nested value and divide to 2
    recycle_dict = {k: v[0] / 2 for k, v in
                    df.loc[df['path'] == RECYCLE, cols_to_divide].reset_index(drop=True).to_dict().items()}
    df.loc[df['path'] == RECYCLE, cols_to_divide] = [recycle_dict[x] for x in cols_to_divide]  # Guarantees the order

    # Calculate totals, after exclusions!
    total = df[["usedlogicalbytes", "maxlogicalbytes", "usedbytes", "maxbytes"]].sum()
    total_row = {
        'Path': 'total',
        'logical used(TB)': "{:,.2f}".format(total["usedlogicalbytes"] / TB_DENOMINATOR),
        'logical quota(TB)': "{:,.2f}".format(total["maxlogicalbytes"] / TB_DENOMINATOR),
        'raw used(TB)': "{:,.2f}".format(total["usedbytes"] / TB_DENOMINATOR),
        'raw quota(TB)': "{:,.2f}".format(total["maxbytes"] / TB_DENOMINATOR),
        'used/quota': "{:,.2f}%".format((total["usedlogicalbytes"] / total["maxlogicalbytes"]) * 100),
        'logical/raw (used)': "{:,.1f}%".format((total["usedlogicalbytes"] / total["usedbytes"]) * 100),
        'logical/raw (quotas)': "{:,.1f}%".format((total["maxlogicalbytes"] / total["maxbytes"]) * 100),
    }

    # Clear inf and nan, also arrange percentage
    df["logical/raw used"] = (df["usedlogicalbytes"] / df["usedbytes"]) * 100
    df["logical/raw used"] = df["logical/raw used"].apply(
        lambda x: "-" if np.isnan(x) or np.isinf(x) else "{:,.1f}%".format(x))
    df["logical/raw quotas"] = (df["maxlogicalbytes"] / df["maxbytes"]) * 100
    df["logical/raw quotas"] = df["logical/raw quotas"].apply(
        lambda x: "-" if np.isnan(x) or np.isinf(x) else "{:,.1f}%".format(x))
    # Arrange percentage of used/quota
    df["percentageusedbytes"] = df["percentageusedbytes"].apply(lambda x: "{:,.2f}%".format(x))

    # Convert to TB
    df["usedlogicalbytes"] = df["usedlogicalbytes"] / TB_DENOMINATOR
    df["maxlogicalbytes"] = df["maxlogicalbytes"] / TB_DENOMINATOR
    df["usedbytes"] = df["usedbytes"] / TB_DENOMINATOR
    df["maxbytes"] = df["maxbytes"] / TB_DENOMINATOR

    # Rename columns
    df = df.rename(columns={
        "path": "Path",
        "usedlogicalbytes": "logical used(TB)",
        "maxlogicalbytes": "logical quota(TB)",
        "usedbytes": "raw used(TB)",
        "maxbytes": "raw quota(TB)",
        "percentageusedbytes": "used/quota",
        "logical/raw used": "logical/raw (used)",
        "logical/raw quotas": "logical/raw (quotas)",
    })
    # Reorder
    df = df[["Path", "logical used(TB)", "logical quota(TB)", "raw used(TB)", "raw quota(TB)", "used/quota",
             "logical/raw (used)", "logical/raw (quotas)",
             ]]
    update_time = get_update_time(input_eos_file)
    html = create_html(df, update_time, total_row)

    with open(output_file, "w") as f:
        f.write(html)


if __name__ == "__main__":
    main()
