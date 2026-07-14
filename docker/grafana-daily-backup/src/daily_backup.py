#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Author: Nikodemas Tuckus <ntuckus AT gmail [DOT] com>
This script copies Grafana dashboard jsons, tars them and puts them into EOS folder
"""
import datetime
import json
import logging
import os
import shutil
import subprocess
import sys
import tarfile

import click
import requests
from helpers.otel_setup import shutdown_opentelemetry

logger = logging.getLogger(__name__)

base_url = "https://monit-grafana.cern.ch/api"
grafana_folder = "./grafana"
tar_file_name = "grafanaBackup.tar.gz"
GRAFANA_ORG_ID = 11
BACKUP_FOLDERS = ["Production", "Development", "Playground", "Backup"]
GENERAL_FOLDER_UID = "general"


def get_grafana_auth(fname):
    if not os.path.exists(fname):
        logger.error("File %s does not exist", fname)
        sys.exit(1)
    with open(fname, "r") as keyFile:
        secret_key = json.load(keyFile).get("filesystem_exporter_token")
    headers = {
        "Authorization": f"Bearer {secret_key}",
        "X-Grafana-Org-Id": str(GRAFANA_ORG_ID),
    }
    return headers


def create_backup_files(dashboard, folder_title, headers):
    all_json = []

    dashboard_uid = dashboard.get("uid")
    path = f"grafana/dashboards/{folder_title}"

    if not os.path.exists(path):
        try:
            os.makedirs(os.path.join("grafana", "dashboards", folder_title))
            logger.info("Successfully created the directory %s", path)
        except OSError:
            logger.error("Creation of the directory %s failed", path)
    dashboard_url = f"{base_url}/dashboards/uid/{dashboard_uid}"
    logger.info("dashboard_url %s", dashboard_url)
    req = requests.get(
        dashboard_url,
        headers=headers,
        params={"orgId": GRAFANA_ORG_ID},
    )
    if req.status_code == 403:
        logger.warning(
            "Skipping dashboard uid %s (%s): forbidden — service account lacks dashboards:read",
            dashboard_uid,
            dashboard.get("title", "unknown"),
        )
        return False
    req.raise_for_status()
    dashboard_data = req.json().get("dashboard")
    if not dashboard_data:
        logger.error("No dashboard data returned for uid %s", dashboard_uid)
        return False

    # Replace characters that might cause harm
    title_of_dashboard = (
        dashboard_data.get("title").replace(" ", "").replace(".", "_").replace("/", "_")
    )
    uid_of_dashboard = dashboard_data.get("uid")

    filename_for_folder = f"{path}/{title_of_dashboard}-{uid_of_dashboard}.json"
    logger.info("filename_for_folder %s", filename_for_folder)
    with open(filename_for_folder, "w+") as jsonFile:
        json.dump(dashboard_data, jsonFile)
    all_json.append(dashboard_data)

    filename_for_all_json = f"{path}/all.json"
    logger.info("filename_for_all_json %s", filename_for_all_json)
    with open(filename_for_all_json, "w+") as jsonFile:
        json.dump(all_json, jsonFile)
    return True


def get_folders(headers):
    response = requests.get(
        f"{base_url}/folders",
        headers=headers,
        params={"orgId": GRAFANA_ORG_ID},
    )
    response.raise_for_status()
    return response.json()


def get_dashboards_in_folder(headers, folder_uid):
    response = requests.get(
        f"{base_url}/search",
        headers=headers,
        params={
            "folderUIDs": folder_uid,
            "type": "dash-db",
            "orgId": GRAFANA_ORG_ID,
        },
    )
    response.raise_for_status()
    return response.json()


def search_folders_from_grafana(headers):
    dashboards_exported = 0
    path_to_db = os.path.abspath("grafana")
    os.makedirs(path_to_db, exist_ok=True)

    folders = get_folders(headers)
    folder_by_title = {folder["title"]: folder for folder in folders}

    for folder_title in BACKUP_FOLDERS:
        folder = folder_by_title.get(folder_title)
        if not folder:
            logger.warning("Folder '%s' not found, skipping", folder_title)
            continue

        folder_id = folder["id"]
        folder_uid = folder["uid"]
        folder_data = get_dashboards_in_folder(headers, folder_uid)

        filename_for_folder = f"grafana/{folder_id}-{folder_title}.json"
        with open(filename_for_folder, "w+") as jsonFile:
            json.dump(folder_data, jsonFile)

        dashboards_amount = len(folder_data)
        for index, dashboard in enumerate(folder_data):
            logger.info("Index/Amount: %s/%s", index, dashboards_amount)
            if create_backup_files(dashboard, folder_title, headers):
                dashboards_exported += 1

    general_dashboards = get_dashboards_in_folder(headers, GENERAL_FOLDER_UID)
    general_amount = len(general_dashboards)
    for index, dashboard in enumerate(general_dashboards):
        logger.info("General Index/Amount: %s/%s", index, general_amount)
        if create_backup_files(dashboard, "General", headers):
            dashboards_exported += 1

    if dashboards_exported == 0:
        logger.error("No dashboards exported, aborting")
        sys.exit(1)


def create_tar(path, tar_name):
    if not os.path.isdir(path) or not any(os.scandir(path)):
        logger.error("Nothing to archive in %s", path)
        sys.exit(1)

    with tarfile.open(tar_name, "w:gz") as tar_handle:
        for root, _, files in os.walk(path):
            for file in files:
                tar_handle.add(os.path.join(root, file))

    tar_size = os.path.getsize(tar_name)
    logger.info("Created archive %s (%d bytes)", tar_name, tar_size)
    if tar_size < 100:
        logger.error("Archive %s is too small (%d bytes), aborting", tar_name, tar_size)
        sys.exit(1)


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
    logger.info("Copy backup to %s", path)
    cmd = f"mkdir -p {path}"
    output = subprocess.check_output(cmd, stderr=subprocess.STDOUT, shell=True)
    if output:
        logger.info(output.decode())
    cmd = f"cp {archive} {path}"
    logger.info("Execute: %s", cmd)
    output = subprocess.check_output(cmd, stderr=subprocess.STDOUT, shell=True)
    if output:
        logger.info(output.decode())
    cmd = f"ls {path}"
    output = subprocess.check_output(cmd, stderr=subprocess.STDOUT, shell=True)
    logger.info("New backup: %s", output.decode())


@click.command()
@click.option(
    "--token", "fname", required=True, help="Grafana token JSON file location"
)
@click.option("--filesystem-path", required=True, help="Path for Grafana backup in EOS")
def main(fname, filesystem_path):
    logger.info("Input Arguments: token:%s, filesystem_path:%s", fname, filesystem_path)
    headers = get_grafana_auth(fname)
    search_folders_from_grafana(headers)
    create_tar(grafana_folder, tar_file_name)
    copy_to_filesystem(tar_file_name, filesystem_path)
    remove_temp_files(grafana_folder)


if __name__ == "__main__":
    try:
        main()
    finally:
        shutdown_opentelemetry()
