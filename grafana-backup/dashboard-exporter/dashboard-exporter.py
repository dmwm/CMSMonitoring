#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Author: Nikodemas Tuckus <ntuckus AT gmail [DOT] com>
This script copies Grafana dashboard jsons, tars them and puts them into EOS folder
"""
import datetime
import json
import os
import shutil
import subprocess
import sys
import tarfile

import click
import requests

base_url = "https://monit-grafana.cern.ch/api"
grafana_folder = "./grafana"
tar_file_name = "grafanaBackup.tar.gz"


def get_grafana_auth(fname):
    if not os.path.exists(fname):
        print(f"File {fname} does not exist")
        sys.exit(1)
    with open(fname, "r") as keyFile:
        secret_key = json.load(keyFile).get("filesystem_exporter_token")
    headers = {"Authorization": f"Bearer {secret_key}"}
    return headers


def create_backup_files(dashboard, folder_title, headers):
    all_json = []

    dashboard_uid = dashboard.get("uid")
    path = f"grafana/dashboards/{folder_title}"

    if not os.path.exists(path):
        try:
            os.makedirs(os.path.join("grafana", "dashboards", folder_title))
            print(f"Successfully created the directory {path}")
        except OSError:
            print(f"Creation of the directory {path} failed")
    dashboard_url = f"{base_url}/dashboards/uid/{dashboard_uid}"
    print("dashboard_url", dashboard_url)
    req = requests.get(dashboard_url, headers=headers)
    dashboard_data = req.json().get("dashboard")

    # Replace characters that might cause harm
    title_of_dashboard = (
        dashboard_data.get("title").replace(" ", "").replace(".", "_").replace("/", "_")
    )
    uid_of_dashboard = dashboard_data.get("uid")

    filename_for_folder = f"{path}/{title_of_dashboard}-{uid_of_dashboard}.json"
    print("filename_for_folder", filename_for_folder)
    with open(filename_for_folder, "w+") as jsonFile:
        json.dump(dashboard_data, jsonFile)
    print("*" * 10 + "\n")
    all_json.append(dashboard_data)

    filename_for_all_json = f"{path}/all.json"
    print("filename_for_all_json", filename_for_all_json)
    with open(filename_for_all_json, "w+") as jsonFile:
        json.dump(all_json, jsonFile)


def search_folders_from_grafana(headers):
    folders_url = f"{base_url}/search?folderIds=0&orgId=11"

    req = requests.get(folders_url, headers=headers)
    if req.status_code != 200:
        print(f"Request {folders_url}, failed with reason: {req.reason}")
        sys.exit(1)
    folders_json = req.json()

    for folder in folders_json:
        folder_id = folder.get("id")
        folder_title = folder.get("title")

        if folder_title not in ["Production", "Development", "Playground", "Backup"]:
            folder_title = "General"
            dashboard = folder
            create_backup_files(dashboard, folder_title, headers)

        else:
            dashboard_query_url = os.path.join(
                base_url, f"search?folderIds={folder_id}&orgId=11&query="
            )

            print("individualFolderUrl", dashboard_query_url)

            req = requests.get(dashboard_query_url, headers=headers)
            folder_data = req.json()

            path_to_db = os.path.abspath("grafana")
            if not os.path.exists(path_to_db):
                os.makedirs(path_to_db)

            filename_for_folder = f"grafana/{folder_id}-{folder_title}.json"
            with open(filename_for_folder, "w+") as jsonFile:
                json.dump(folder_data, jsonFile)

            dashboards_amount = len(folder_data)
            for index, dashboard in enumerate(folder_data):
                folder_title = dashboard.get("folderTitle")
                print(f"Index/Amount: {index}/{dashboards_amount}")

                create_backup_files(dashboard, folder_title, headers)


def create_tar(path, tar_name):
    with tarfile.open(tar_name, "w:gz") as tar_handle:
        for root, _, files in os.walk(path):
            for file in files:
                tar_handle.add(os.path.join(root, file))


def remove_temp_files(path):
    for root, dirs, files in os.walk(path):
        for f in files:
            os.unlink(os.path.join(root, f))
        for d in dirs:
            shutil.rmtree(os.path.join(root, d))
    if os.path.exists(path):
        os.rmdir(path)


def get_date():
    dt = datetime.datetime.now().strftime("%Y/%m/%d")
    return dt


def copy_to_filesystem(archive, base_dir):
    dt = get_date()
    path = os.path.join(base_dir, dt)
    print(f"Copy backup to {path}")
    cmd = f"mkdir -p {path}"
    output = subprocess.check_output(cmd, stderr=subprocess.STDOUT, shell=True)
    if output:
        print(output)
    cmd = f"cp {archive} {path}"
    print(f"Execute: {cmd}")
    output = subprocess.check_output(cmd, stderr=subprocess.STDOUT, shell=True)
    if output:
        print(output)
    cmd = f"ls {path}"
    output = subprocess.check_output(cmd, stderr=subprocess.STDOUT, shell=True)
    print(f"New backup: {output}")


@click.command()
@click.option(
    "--token", "fname", required=True, help="Grafana token JSON file location"
)
@click.option("--filesystem-path", required=True, help="Path for Grafana backup in EOS")
def main(fname, filesystem_path):
    click.echo(f"Input Arguments: token:{fname}, filesystem_path:{filesystem_path}")
    headers = get_grafana_auth(fname)
    search_folders_from_grafana(headers)
    create_tar(grafana_folder, tar_file_name)
    copy_to_filesystem(tar_file_name, filesystem_path)
    remove_temp_files(grafana_folder)


if __name__ == "__main__":
    main()
