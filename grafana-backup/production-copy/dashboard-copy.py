#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
This script copies Grafana dashboards from one folder to another.
"""
import datetime
import json
import logging
import sys

import click
import requests

logging.basicConfig(level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s")


def get_grafana_auth(fname):
    """Load Grafana authentication token from a file."""
    with open(fname, "r") as token_file:
        token = json.load(token_file)["production_copy_token"]
    return {"Authorization": f"Bearer {token}", "Content-Type": "application/json"}


def get_folder_id(base_url, headers, folder_name):
    """Fetch the folder ID for a given folder name."""
    response = requests.get(f"{base_url}/api/folders", headers=headers)
    response.raise_for_status()
    for folder in response.json():
        if folder["title"] == folder_name:
            return folder["id"]
    return None


def get_folder_uid(base_url, headers, folder_name):
    """Fetch the folder UID for a given folder name."""
    response = requests.get(f"{base_url}/api/folders", headers=headers)
    response.raise_for_status()
    for folder in response.json():
        if folder["title"] == folder_name:
            return folder["uid"]
    return None


def get_folder_permissions(base_url, headers, folder_uid):
    """Fetch the permissions of a folder."""
    response = requests.get(f"{base_url}/api/folders/{folder_uid}/permissions", headers=headers)
    response.raise_for_status()
    return response.json()


def update_folder_permissions(base_url, headers, folder_uid, permissions):
    """Set permissions for a folder."""
    payload = {"items": permissions}
    response = requests.post(f"{base_url}/api/folders/{folder_uid}/permissions", headers=headers, json=payload)
    response.raise_for_status()
    logging.info(f"Permissions for folder UID '{folder_uid}' have been updated.")


def delete_folder(base_url, headers, folder_uid):
    """Delete a folder by its UID."""
    response = requests.delete(f"{base_url}/api/folders/{folder_uid}", headers=headers)
    response.raise_for_status()
    logging.info(f"Deleted folder with UID {folder_uid}.")


def create_folder(base_url, headers, folder_name):
    """Create a folder. If it exists, delete and recreate it while preserving permissions."""
    folder_uid = get_folder_uid(base_url, headers, folder_name)
    preserved_permissions = []

    if folder_uid:
        logging.info(f"Folder '{folder_name}' exists. Fetching permissions and overwriting...")
        preserved_permissions = get_folder_permissions(base_url, headers, folder_uid)
        delete_folder(base_url, headers, folder_uid)

    response = requests.post(f"{base_url}/api/folders", headers=headers, json={"title": folder_name})
    response.raise_for_status()
    folder = response.json()
    logging.info(f"Created folder '{folder_name}'.")

    if preserved_permissions:
        update_folder_permissions(base_url, headers, folder["uid"], preserved_permissions)

    return folder["id"]


def get_dashboards_in_folder(base_url, headers, folder_id):
    """Fetch all dashboards in a folder."""
    response = requests.get(f"{base_url}/api/search?folderIds={folder_id}&type=dash-db", headers=headers)
    response.raise_for_status()
    return response.json()


def copy_dashboard(base_url, headers, dashboard, target_folder_id):
    """Copy a dashboard to a new folder."""
    dashboard_uid = dashboard["uid"]
    response = requests.get(f"{base_url}/api/dashboards/uid/{dashboard_uid}", headers=headers)
    response.raise_for_status()
    dashboard_data = response.json()
    dashboard_data["dashboard"]["id"] = None
    dashboard_data["dashboard"]["uid"] = None
    dashboard_data["folderId"] = target_folder_id
    dashboard_data["message"] = f"Copied on {datetime.datetime.now()}"
    response = requests.post(f"{base_url}/api/dashboards/db", headers=headers, json=dashboard_data)
    response.raise_for_status()
    logging.info(f"Copied dashboard '{dashboard['title']}' to folder ID {target_folder_id}.")


@click.command()
@click.option("--url", required=True, help="Base URL of the Grafana instance")
@click.option("--token", "fname", required=True, help="API or Service Account token for authentication")
def main(url, fname):
    headers = get_grafana_auth(fname)
    source_folder_name = "Production"
    target_folder_name = "Production Copy"

    source_folder_id = get_folder_id(url, headers, source_folder_name)
    if not source_folder_id:
        logging.error(f"Source folder '{source_folder_name}' does not exist.")
        return

    target_folder_id = create_folder(url, headers, target_folder_name)
    dashboards = get_dashboards_in_folder(url, headers, source_folder_id)
    logging.info(f"Found {len(dashboards)} dashboards in '{source_folder_name}'.")

    for dashboard in dashboards:
        copy_dashboard(url, headers, dashboard, target_folder_id)


if __name__ == "__main__":
    try:
        main()
    except Exception as e:
        logging.error(f"An unexpected error occurred: {e}")
        sys.exit(1)
