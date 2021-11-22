# !/usr/bin/env python
# -*- coding: utf-8 -*-
# Author: Ceyhun Uzunoglu <ceyhunuzngl AT gmail [DOT] com>
# Create html table for EOS paths' size as a result of xrdcp command

# acronjob: $HOME/CMSMonitoring/scripts/eos_path_size.sh

import json
import sys

import click
import numpy as np
import pandas as pd
from datetime import datetime

pd.options.display.float_format = "{:,.2f}".format
pd.set_option("display.max_colwidth", -1)

EXCLUDED_PATHS = ["/eos/cms/store/cmst3", "/eos/recovered", "/eos/totem"]
RECYCLE = "/eos/cms/proc/recycle/"
TB_DENOMINATOR = 10 ** 12


@click.command()
@click.option("--output_folder", default=None, required=True, help="For example: /eos/.../www/test/test.html")
def main(output_folder=None):
    # Get xrdcp command output as input to python script.
    data = json.load(sys.stdin)

    df = pd.DataFrame(data['storageservice']['storageshares'])
    df = df[["path", "usedsize", "totalsize"]]
    df = df.rename(columns={
        "usedsize": "logical used size",
        "totalsize": "logical quota",
    })

    # Convert path list to string
    df["path"] = df["path"].apply(lambda x: ",".join(x))
    # Filter out cmst3/*
    df = df[~df["path"].str.contains("|".join(EXCLUDED_PATHS), regex=True)]

    # RECYCLE: divide to 2
    df['logical used size'] = df.apply(
        lambda x: x["logical used size"] / 2.0 if (x.path == RECYCLE) else x["logical used size"],
        axis=1
    )
    df['logical quota'] = df.apply(
        lambda x: x["logical quota"] / 2.0 if (x.path == RECYCLE) else x["logical quota"],
        axis=1
    )

    # Calculate raw sizes
    df['raw used size'] = df['logical used size'] * 2.0
    df['raw quota'] = df['logical quota'] * 2.0

    # Calculate totals, after exclusions!
    total = df[["logical used size", "logical quota", "raw used size", "raw quota"]].sum()
    total_row = {
        'path': 'Total',
        'logical used size': total["logical used size"] / TB_DENOMINATOR,
        'logical quota': total["logical quota"] / TB_DENOMINATOR,
        'raw used size': total["raw used size"] / TB_DENOMINATOR,
        'raw quota': total["raw quota"] / TB_DENOMINATOR,
        'used/total': "{:,.1f}%".format((total["logical used size"] / total["logical quota"]) * 100)
    }
    df["used/total"] = (df["logical used size"] / df["logical quota"]) * 100
    # Clear inf and nan, also arrange percentage
    df["used/total"] = df["used/total"].apply(lambda x: "-" if np.isnan(x) or np.isinf(x) else "{:,.1f}%".format(x))

    df["logical used size"] = df["logical used size"] / TB_DENOMINATOR
    df["logical quota"] = df["logical quota"] / TB_DENOMINATOR
    df["raw used size"] = df["raw used size"] / TB_DENOMINATOR
    df["raw quota"] = df["raw quota"] / TB_DENOMINATOR

    df = df.append(total_row, ignore_index=True)

    df = df.rename(columns={
        "logical used size": "logical used size(TB)",
        "logical quota": "logical quota(TB)",
        "raw used size": "raw used size(TB)",
        "raw quota": "raw quota(TB)",
        "used/total": "used/quota",
    })

    main_column = df["path"].copy()
    df["path"] = (
        f'<a class="path">'
        + main_column
        + '</a><br>'
    )

    html = df.to_html(escape=False, index=False)
    html = html.replace(
        'table border="1" class="dataframe"',
        'table id="dataframe" class="display compact" style="width:100%;"',
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
                <small>Last Update: {datetime.utcnow().strftime("%Y-%m-%d %H:%M:%S")} UTC</small>
            </h1>
        </div>
        <div class="w3-container">
        <ul class="w3-ul w3-small" style="width:70%;margin-left:0%">
          <li class="w3-padding-small">
            This page shows the size of CMS EOS paths in TB. Bytes to TB denominator is <b>10^12</b>.
          </li>
          <li class="w3-padding-small">
            Used command to get values: <b><code>xrdcp root://eoscms.cern.ch//eos/cms/proc/accounting -</b>
          </li>
          <li class="w3-padding-small">
            Acronjob: <b><code>xrdcp root://eoscms.cern.ch//eos/cms/proc/accounting - |
            python eos_path_size.py --output_folder=/eos/user/c/cmsmonit/www/eos-path-size/size.html</b>
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
                    "order": [[ 1, "desc" ]],
                    "pageLength" : 200,
                    "scrollX": false,
                    language: {
                        search: "_INPUT_",
                        searchPlaceholder: "--- Search EOS Path ---",
                    },
                });
            });
        </script>
    </body>
    </html>'''
    )

    result = html_header + html_middle + html + html_footer

    with open(output_folder, "w") as f:
        f.write(result)


if __name__ == "__main__":
    main()
