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


def get_paths_from_xrdcp(json_data):
    """Get paths from XRDCP command and filter our cmst3 paths """
    data = json.load(json_data)
    paths = [elem["path"][0] for elem in data['storageservice']['storageshares']]
    # Filter out cmst3/*
    paths = [path for path in paths if all((exc not in path) for exc in EXCLUDED_PATHS)]
    return paths


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


def get_html_template(base_html_directory=None):
    """ Reads partial html file and return it as strings
    """
    if base_html_directory is None:
        base_html_directory = os.getcwd()
    with open(os.path.join(base_html_directory, "main.html")) as f:
        main_html = f.read()
    return main_html


def create_html(df, update_time, total_row, base_html_directory):
    """Create html page with given dataframe
    """
    # Get main html
    main_html = get_html_template(base_html_directory=base_html_directory)
    main_html = main_html.replace("__UPDATE_TIME__", update_time)
    main_html = main_html.replace("__EXCLUDED_PATHS__", "*,".join(EXCLUDED_PATHS) + "*")
    # Total row placeholder
    total_on_header = """
                <tr >
                    <th>{Path}</th>
                    <th>{logical used(TB)}</th>
                    <th>{logical quota(TB)}</th>
                    <th>{raw used(TB)}</th>
                    <th>{raw quota(TB)}</th>
                    <th>{used/quota}</th>
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
    # Pandas df to html
    html = df.to_html(escape=False, index=False)
    # Add total ro to header. Be careful there should be only one  <thead>...</thead>!
    html = html.replace(" </thead>", total_on_header)
    # cleanup of the default dump
    html = html.replace(
        'table border="1" class="dataframe"',
        'table id="dataframe" class="display compact" style="width:100%;"',
    )
    html = html.replace('style="text-align: right;"', "")

    # Add pandas dataframe html to main body
    main_html = main_html.replace("____MAIN_BLOCK____", html)
    return main_html


@click.command()
@click.option("--output_file", default=None, required=True, help="For example: /eos/.../www/test/test.html")
@click.option("--input_eos_file", default=None, required=False,
              help="Result of 'eos -r 103074 1399 quota ls -m', i.e.: /eos/cms/store/accounting/eos_quota_ls.txt")
@click.option("--static_html_dir", default=None, required=True,
              help="Html directory for main html template. For example: ~/CMSMonitoring/src/html/eos_path_size")
def main(output_file=None, input_eos_file=None, static_html_dir=None):
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
        'logical/raw (quotas)': "{:,.1f}%".format((total["maxlogicalbytes"] / total["maxbytes"]) * 100),
    }

    # Clear inf and nan, also arrange percentage
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
        "logical/raw quotas": "logical/raw (quotas)",
    })
    # Reorder
    df = df[["Path", "logical used(TB)", "logical quota(TB)", "raw used(TB)", "raw quota(TB)", "used/quota",
             "logical/raw (quotas)"]]
    update_time = get_update_time(input_eos_file)
    html = create_html(df=df,
                       update_time=update_time,
                       total_row=total_row,
                       base_html_directory=static_html_dir)

    with open(output_file, "w") as f:
        f.write(html)


if __name__ == "__main__":
    main()
