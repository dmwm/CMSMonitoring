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
from helpers.otel_setup import shutdown_opentelemetry

logger = logging.getLogger(__name__)

GRAFANA_ORG_ID = 11


def get_grafana_auth(fname):
    """Load Grafana authentication token from a file."""
    with open(fname, "r") as token_file:
        token = json.load(token_file)["production_copy_token"]
    return {"Authorization": f"Bearer {token}", "Content-Type": "application/json"}


def get_folder_uid(base_url, headers, folder_name):
    """Fetch the folder UID for a given folder name."""
    response = requests.get(
        f"{base_url}/api/folders",
        headers=headers,
        params={"orgId": GRAFANA_ORG_ID},
    )
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
    logger.info("Permissions for folder UID '%s' have been updated.", folder_uid)


def delete_folder(base_url, headers, folder_uid):
    """Delete a folder by its UID."""
    response = requests.delete(f"{base_url}/api/folders/{folder_uid}", headers=headers)
    response.raise_for_status()
    logger.info("Deleted folder with UID %s.", folder_uid)


def create_folder(base_url, headers, folder_name):
    """Create a folder. If it exists, delete and recreate it while preserving permissions."""
    folder_uid = get_folder_uid(base_url, headers, folder_name)
    preserved_permissions = []

    if folder_uid:
        logger.info("Folder '%s' exists. Fetching permissions and overwriting...", folder_name)
        preserved_permissions = get_folder_permissions(base_url, headers, folder_uid)
        delete_folder(base_url, headers, folder_uid)

    response = requests.post(f"{base_url}/api/folders", headers=headers, json={"title": folder_name})
    response.raise_for_status()
    folder = response.json()
    logger.info("Created folder '%s'.", folder_name)

    if preserved_permissions:
        update_folder_permissions(base_url, headers, folder["uid"], preserved_permissions)

    return folder["id"]


def get_dashboards_in_folder(base_url, headers, folder_uid):
    """Fetch all dashboards in a folder."""
    response = requests.get(
        f"{base_url}/api/search",
        headers=headers,
        params={"folderUIDs": folder_uid, "type": "dash-db", "orgId": GRAFANA_ORG_ID},
    )
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
    logger.info("Copied dashboard '%s' to folder ID %s.", dashboard["title"], target_folder_id)


@click.command()
@click.option("--url", required=True, help="Base URL of the Grafana instance")
@click.option("--token", "fname", required=True, help="API or Service Account token for authentication")
def main(url, fname):
    headers = get_grafana_auth(fname)
    source_folder_name = "Production"
    target_folder_name = "Production Copy"

    source_folder_uid = get_folder_uid(url, headers, source_folder_name)
    if not source_folder_uid:
        logger.error("Source folder '%s' does not exist.", source_folder_name)
        return

    dashboards = get_dashboards_in_folder(url, headers, source_folder_uid)
    logger.info("Found %s dashboards in '%s'.", len(dashboards), source_folder_name)
    if not dashboards:
        logger.error(
            "No dashboards found in '%s'. Aborting without modifying '%s'.",
            source_folder_name,
            target_folder_name,
        )
        return

    target_folder_id = create_folder(url, headers, target_folder_name)

    for dashboard in dashboards:
        copy_dashboard(url, headers, dashboard, target_folder_id)


if __name__ == "__main__":
    try:
        main()
    except Exception as e:
        logger.error("An unexpected error occurred: %s", e)
        sys.exit(1)
    finally:
        shutdown_opentelemetry()
