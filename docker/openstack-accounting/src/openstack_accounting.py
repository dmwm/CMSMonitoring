# !/usr/bin/env python
# -*- coding: utf-8 -*-
# Author: Ceyhun Uzunoglu <ceyhunuzngl AT gmail [DOT] com>
# Create html table for Openstack Projects accounting
#
# cron job script    : scripts/cron4openstack_accounting.sh
# Kubernetes service : https://github.com/dmwm/CMSKubernetes/blob/master/kubernetes/monitoring/services/cron-size-quotas.yaml
#

import json
import os
import sys
from datetime import datetime

import click
import pandas as pd
from schema import Schema, Use, SchemaError

pd.options.display.float_format = "{:,.2f}".format
pd.set_option("display.max_colwidth", None)
total_row_openstackProjectName = "TOTAL"

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
                          'totalBackupGigabytesUsed': Use(int),
                          'contacts': str, }])

# DO NOT FORGET TO UPDATE "columnDefs" VISIBILITY IN JavaScript TEMPLATE: src/html/openstack_accounting/main.html
# ORDER IS IMPORTANT IN PYTHON DICTS AND IT IS USED IN JS "columnDefs" ARRAY
SUMMARY_COL_ORDER = {'openstackProjectName': 'Openstack Project Name',
                     'maxTotalInstances': 'Max Total Instances',
                     'maxTotalCores': 'Max Total Cores',
                     'maxTotalRAMSize': 'Max Total RAM Size',
                     'maxSecurityGroups': 'Max Security Groups',
                     'maxTotalFloatingIps': 'Max Total Floating Ips',
                     'maxServerMeta': 'Max Server Meta',
                     'maxImageMeta': 'Max Image Meta',
                     'maxPersonality': 'Max Personality',
                     'maxPersonalitySize': 'Max Personality Size',
                     'maxSecurityGroupRules': 'Max Security Group Rules',
                     'maxTotalKeypairs': 'Max Total Keypairs',
                     'maxServerGroups': 'Max Server Groups',
                     'maxServerGroupMembers': 'Max Server Group Members',
                     'totalRAMUsed': 'Total RAM Used',
                     'totalCoresUsed': 'Total Cores Used',
                     'totalInstancesUsed': 'Total Instances Used',
                     'totalFloatingIpsUsed': 'Total Floating Ips Used',
                     'totalSecurityGroupsUsed': 'Total Security Groups Used',
                     'totalServerGroupsUsed': 'Total Server Groups Used',
                     'maxTotalVolumes': 'Max Total Volumes',
                     'maxTotalSnapshots': 'Max Total Snapshots',
                     'maxTotalVolumeGigabytes': 'Max Total Volume Gigabytes',
                     'maxTotalBackups': 'Max Total Backups',
                     'maxTotalBackupGigabytes': 'Max Total Backup Gigabytes',
                     'totalVolumesUsed': 'Total Volumes Used',
                     'totalGigabytesUsed': 'Total Gigabytes Used',
                     'totalSnapshotsUsed': 'Total Snapshots Used',
                     'totalBackupsUsed': 'Total Backups Used',
                     'totalBackupGigabytesUsed': 'Total Backup Gigabytes Used',
                     'contacts': 'Contacts'}


def tstamp():
    """Return timestamp for logging"""
    return datetime.utcnow().strftime('%Y-%m-%d %H:%M:%S')


def get_update_time_of_file(summary_file):
    """Create update time depending on reading Openstack accounting results from file"""
    if summary_file:
        # Set update time to Openstack accounting file modification time, minimum of 2
        summary_ts = os.path.getmtime(summary_file)
        try:
            return datetime.utcfromtimestamp(summary_ts).strftime('%Y-%m-%d %H:%M:%S')
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
        df = pd.DataFrame(json_arr, columns=column_order.keys()).rename(columns=column_order)

        # Find the dict with openstackProjectName= "TOTAL" in JSON array
        total_dict = next(
            (sub for sub in json_arr if sub['openstackProjectName'] == total_row_openstackProjectName),
            None
        )
        # remove TOTAL row from json array
        json_arr.remove(total_dict)

        # Create 2 level columns dataframe to make total row to fit the top like column names.
        columns = list(zip(SUMMARY_COL_ORDER.values(), total_dict.values()))
        df.columns = pd.MultiIndex.from_tuples(columns)

        return df
    except SchemaError as e:
        print(tstamp(), "Data not exist or not valid:", str(e))
        sys.exit(1)
    except json.JSONDecodeError as e:
        print(tstamp(), json_file, "file is empty:", str(e))
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
@click.option("--summary_json", required=True, help="/eos/cms/store/accounting/openstack_accounting_summary.json")
@click.option("--static_html_dir", default=None, required=True,
              help="Html directory for main html template. For example: ~/CMSMonitoring/src/html/openstack_accounting")
def main(output_file=None, summary_json=None, static_html_dir=None):
    joint_update_time = get_update_time_of_file(summary_json)
    print("[INFO] Update time of input files:", joint_update_time)

    df_summary = get_df_with_validation(summary_json, SUMMARY_SCHEMA, SUMMARY_COL_ORDER)
    main_html = create_main_html(df_summary, joint_update_time, static_html_dir)
    with open(output_file, "w+") as f:
        f.write(main_html)


if __name__ == "__main__":
    main()
