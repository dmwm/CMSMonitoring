# !/usr/bin/env python
# -*- coding: utf-8 -*-
# Author: Ceyhun Uzunoglu <ceyhunuzngl AT gmail [DOT] com>
# Create html table for EOS paths' size
#
# acronjob: $HOME/CMSMonitoring/scripts/eos_path_size.sh
#

import json
import os
import sys
from datetime import datetime

import click
import pandas as pd
from schema import Schema, Use, SchemaError, Or

pd.options.display.float_format = "{:,.2f}".format
pd.set_option("display.max_colwidth", None)

SUMMARY_SCHEMA = Schema([{'path': str,
                         'usedterabytes': Use(float),
                         'usedlogicalterabytes': Use(float),
                         'maxphysicalterabytes': Use(float),
                         'maxlogicalterabytes': Use(float),
                         'used_logical_space_percentage': Or(float, int, None),
                         'used_logical_over_used_raw_percentage': Or(float, int, None), }])

SUMMARY_COL_ORDER = {'path': 'Path',
                    'usedlogicalterabytes': 'Used [TB] logical ',
		    'usedterabytes': 'Used [TB] physical',
		    'maxlogicalterabytes': 'Logical quota [TB] ',
                    'maxphysicalterabytes': 'Physical quota [TB]',
                    'used_logical_space_percentage': '% Logical used',
                    'used_logical_over_used_raw_percentage': '% Used logical / Used physical'}

NON_EC_SCHEMA = Schema([{'path': str,
                         'usedfiles': Use(int),
                         'usedterabytes': Use(float),
                         'usedlogicalterabytes': Use(float),
                         'maxterabytes': Use(float),
                         'maxlogicalterabytes': Use(float),
                         'percentageusedterabytes': Use(float),
                         'used_logical_over_used_raw_percentage': Or(float, int, None),
                         'quota': str,
                         'gid': Use(str),
                         'maxfiles': Use(int),
                         'statusbytes': str,
                         'statusfiles': str, }])

NON_EC_COL_ORDER = {'path': 'Path',
                    'usedfiles': 'Used Files',
                    'usedterabytes': 'Used TB',
                    'usedlogicalterabytes': 'Used Logical TB',
                    'maxterabytes': 'Max TB',
                    'maxlogicalterabytes': 'Max Logical TB',
                    'percentageusedterabytes': 'Percentage Used %',
                    'used_logical_over_used_raw_percentage': '% Used Logical / Used Raw',
                    'quota': 'Quota',
                    'gid': 'GID',
                    'maxfiles': 'Max Files',
                    'statusbytes': 'Status Bytes',
                    'statusfiles': 'Status Files'}

EC_SCHEMA = Schema([{'quota_node': str,
                     'max_logical_quota': Use(float),
                     'free_physical': Use(float),
                     'free_physical_for_ec': Use(float),
                     'free_physical_for_rep': Use(float),
                     'free_logical': Use(float),
                     'total_used_logical_terabytes': Use(float),
                     'used_logical_over_used_raw_percentage': Or(float, int, None),
                     'logical_rep_terabytes': Use(float),
                     'logical_ec_terabytes': Use(float),
                     'max_physical_quota': Use(float),
                     'total_used_physical_terabytes': Use(float),
                     'physical_rep_terabytes': Use(float),
                     'physical_ec_terabytes': Use(float), }])

EC_COL_ORDER = {'quota_node': 'Path',
                'free_logical': 'Free Logical',
                'total_used_logical_terabytes': 'Total Used Logical TB',
                'max_logical_quota': 'Logical Quota',
                'max_physical_quota': 'Physical Quota',
                'free_physical': 'Free Physical',
                'total_used_physical_terabytes': 'Total Used Physical TB',
                'used_logical_over_used_raw_percentage': 'Used Logical / Used Raw %',
                'free_physical_for_ec': 'Free Physical EC',
                'free_physical_for_rep': 'Free Physical Rep',
                'logical_rep_terabytes': 'Logical Rep TB',
                'logical_ec_terabytes': 'Logical EC TB',
                'physical_rep_terabytes': 'Physical Rep TB',
                'physical_ec_terabytes': 'Physical EC TB', }


def tstamp():
    """Return timestamp for logging"""
    return datetime.utcnow().strftime('%Y-%m-%d %H:%M:%S')


def get_update_time_of_file(ec_file, non_ec_file, summary_file):
    """Create update time depending on reading EOS results from file or directly from command"""
    if ec_file and non_ec_file and summary_file:
        # Set update time to eos file modification time, minimum of 2
        ec_ts = os.path.getmtime(ec_file)
        non_ec_ts = os.path.getmtime(non_ec_file)
        summary_ts = os.path.getmtime(summary_file)
        try:
            return datetime.utcfromtimestamp(min(ec_ts, non_ec_ts, summary_ts)).strftime('%Y-%m-%d %H:%M:%S')
        except OSError as e:
            print(tstamp(), "ERROR: could not get last modification time of file:", str(e))
    else:
        # !! means time did not come from file but cron job time
        return "!!" + datetime.utcnow().strftime("%Y-%m-%d H:%M:%S")


def get_df_with_validation(json_file, schema, column_order):
    """Read json file, validate, cast types and convert to pandas dataframe
    """
    try:
        with open(json_file) as f:
            json_arr = json.load(f)

        json_arr = schema.validate(json_arr)

        # orient values reads json array
        return pd.DataFrame(json_arr, columns=column_order.keys()).rename(columns=column_order)
    except SchemaError as e:
        print(tstamp(), "Data not exist or not valid:", str(e))
        sys.exit(1)


def get_html_template(base_html_directory=None):
    """ Reads partial html file and return it as strings
    """
    if base_html_directory is None:
        base_html_directory = os.getcwd()
    with open(os.path.join(base_html_directory, "main.html")) as f:
        main_html = f.read()
    return main_html


def prepare_html(df):
    html = df.to_html(escape=False, index=False)
    # cleanup of the default dump
    html = html.replace(
        'table border="1" class="dataframe"',
        'table id="" class="display compact" style="width:90%;"',
    )
    html = html.replace('style="text-align: right;"', "")
    return html


def create_main_html(df_non_ec, df_ec, df_summary, update_time, base_html_directory):
    """Create html page with given dataframe
    """
    df_non_ec_html = prepare_html(df_non_ec)
    df_ec_html = prepare_html(df_ec)
    df_summary_html = prepare_html(df_summary)

    # Get main html
    main_html = get_html_template(base_html_directory=base_html_directory)
    main_html = main_html.replace("__UPDATE_TIME__", update_time)

    # Add pandas dataframe html to main body
    main_html = main_html.replace('____NON_EC_BLOCK____', df_non_ec_html)
    main_html = main_html.replace('____EC_BLOCK____', df_ec_html)
    main_html = main_html.replace('____SUMMARY_BLOCK____', df_summary_html)
    return main_html


@click.command()
@click.option("--output_file", default=None, required=True, help="For example: /eos/.../www/test/test.html")
@click.option("--non_ec_json", required=True, help="/eos/cms/store/accounting/eos_non_ec_accounting.json")
@click.option("--ec_json", required=True, help="/eos/cms/store/accounting/eos_ec_accounting.json")
@click.option("--summary_json", required=True, help="/eos/cms/store/eos_accounting_summary.json")
@click.option("--static_html_dir", default=None, required=True,
              help="Html directory for main html template. For example: ~/CMSMonitoring/src/html/eos_path_size")
def main(output_file=None, non_ec_json=None, ec_json=None, summary_json=None, static_html_dir=None):
    """Main function combines xrdcp and EOS results then creates HTML page
    """
    joint_update_time = get_update_time_of_file(non_ec_json, ec_json, summary_json)
    print("[INFO] Update time of input files:", joint_update_time)

    df_non_ec = get_df_with_validation(non_ec_json, NON_EC_SCHEMA, NON_EC_COL_ORDER)
    df_ec = get_df_with_validation(ec_json, EC_SCHEMA, EC_COL_ORDER)
    df_summary = get_df_with_validation(summary_json, SUMMARY_SCHEMA, SUMMARY_COL_ORDER)
    main_html = create_main_html(df_non_ec, df_ec, df_summary, joint_update_time, static_html_dir)
    with open(output_file, "w") as f:
        f.write(main_html)


if __name__ == "__main__":
    main()
