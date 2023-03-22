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

SUMMARY_SCHEMA = Schema([{'openstackProjectName': str,
                          'maxTotalInstances': Use(int),
                          'maxTotalCores': Use(int),
                          'maxTotalRAMSize': Use(int),
                          'maxSecurityGroups': Use(int),
                          'maxTotalFloatingIps': Use(int),
                          'maxServerMeta': Use(int),
                          'maxImageMeta': Use(int),
                          'maxPersonality': Use(int),
                          'maxPersonalitySize': Use(int),
                          'maxSecurityGroupRules': Use(int),
                          'maxTotalKeypairs': Use(int),
                          'maxServerGroups': Use(int),
                          'maxServerGroupMembers': Use(int),
                          'totalRAMUsed': Use(int),
                          'totalCoresUsed': Use(int),
                          'totalInstancesUsed': Use(int),
                          'totalFloatingIpsUsed': Use(int),
                          'totalSecurityGroupsUsed': Use(int),
                          'totalServerGroupsUsed': Use(int),
                          'maxTotalVolumes': Use(int),
                          'maxTotalSnapshots': Use(int),
                          'maxTotalVolumeGigabytes': Use(int),
                          'maxTotalBackups': Use(int),
                          'maxTotalBackupGigabytes': Use(int),
                          'totalVolumesUsed': Use(int),
                          'totalGigabytesUsed': Use(int),
                          'totalSnapshotsUsed': Use(int),
                          'totalBackupsUsed': Use(int),
                          'totalBackupGigabytesUsed': Use(int),}])


SUMMARY_COL_ORDER = {'openstackProjectName': 'Openstack Project Name',
                     'maxTotalInstances': 'maxTotalInstances',
                     'maxTotalCores': 'maxTotalCores',
                     'maxTotalRAMSize': 'maxTotalRAMSize',
                     'maxSecurityGroups': 'maxSecurityGroups',
                     'maxTotalFloatingIps': 'maxTotalFloatingIps',
                     'maxServerMeta': 'maxServerMeta',
                     'maxImageMeta': 'maxImageMeta',
                     'maxPersonality': 'maxPersonality',
                     'maxPersonalitySize': 'maxPersonalitySize',
                     'maxSecurityGroupRules': 'maxSecurityGroupRules',
                     'maxTotalKeypairs': 'maxTotalKeypairs',
                     'maxServerGroups': 'maxServerGroups',
                     'maxServerGroupMembers': 'maxServerGroupMembers',
                     'totalRAMUsed': 'totalRAMUsed',
                     'totalCoresUsed': 'totalCoresUsed',
                     'totalInstancesUsed': 'totalInstancesUsed',
                     'totalFloatingIpsUsed': 'totalFloatingIpsUsed',
                     'totalSecurityGroupsUsed': 'totalSecurityGroupsUsed',
                     'totalServerGroupsUsed': 'totalServerGroupsUsed',
                     'maxTotalVolumes': 'maxTotalVolumes',
                     'maxTotalSnapshots': 'maxTotalSnapshots',
                     'maxTotalVolumeGigabytes': 'maxTotalVolumeGigabytes',
                     'maxTotalBackups': 'maxTotalBackups',
                     'maxTotalBackupGigabytes': 'maxTotalBackupGigabytes',
                     'totalVolumesUsed': 'totalVolumesUsed',
                     'totalGigabytesUsed': 'totalGigabytesUsed',
                     'totalSnapshotsUsed': 'totalSnapshotsUsed',
                     'totalBackupsUsed': 'totalBackupsUsed',
                     'totalBackupGigabytesUsed': 'totalBackupGigabytesUsed'}

def tstamp():
    """Return timestamp for logging"""
    return datetime.utcnow().strftime('%Y-%m-%d %H:%M:%S')


def get_update_time_of_file(summary_file):
    """Create update time depending on reading EOS results from file or directly from command"""
    if summary_file:
        # Set update time to eos file modification time, minimum of 2
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


def create_main_html(df_summary, update_time, base_html_directory):
    """Create html page with given dataframe
    """
    df_summary_html = prepare_html(df_summary)

    # Get main html
    main_html = get_html_template(base_html_directory=base_html_directory)
    main_html = main_html.replace("__UPDATE_TIME__", update_time)

    # Add pandas dataframe html to main body
    main_html = main_html.replace('____SUMMARY_BLOCK____', df_summary_html)
    return main_html


@click.command()
@click.option("--output_file", default=None, required=True, help="For example: /eos/.../www/test/test.html")
@click.option("--summary_json", required=True, help="/eos/cms/store/eos_accounting_summary.json")
@click.option("--static_html_dir", default=None, required=True,
              help="Html directory for main html template. For example: ~/CMSMonitoring/src/html/eos_path_size")
def main(output_file=None, non_ec_json=None, ec_json=None, summary_json=None, static_html_dir=None):
    """Main function combines xrdcp and EOS results then creates HTML page
    """
    joint_update_time = get_update_time_of_file(non_ec_json, ec_json, summary_json)
    print("[INFO] Update time of input files:", joint_update_time)

    df_summary = get_df_with_validation(summary_json, SUMMARY_SCHEMA, SUMMARY_COL_ORDER)
    main_html = create_main_html(df_non_ec, df_ec, df_summary, joint_update_time, static_html_dir)
    with open(output_file, "w") as f:
        f.write(main_html)


if __name__ == "__main__":
    main()
