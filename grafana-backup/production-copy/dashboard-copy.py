import datetime
import logging

import click
import requests

logging.basicConfig(level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s")


def get_folder_id(base_url, headers, folder_name):
    """Fetch the folder ID for a given folder name."""
    response = requests.get(f"{base_url}/api/folders", headers=headers)
    response.raise_for_status()
    for folder in response.json():
        if folder["title"] == folder_name:
            return folder["id"]
    return None


def create_folder(base_url, headers, folder_name):
    """Create a folder if it doesn't already exist."""
    folder_id = get_folder_id(base_url, headers, folder_name)
    if folder_id:
        logging.info(f"Folder '{folder_name}' already exists.")
        return folder_id
    response = requests.post(f"{base_url}/api/folders", headers=headers, json={"title": folder_name})
    response.raise_for_status()
    folder = response.json()
    logging.info(f"Created folder '{folder_name}'.")
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
@click.option("--token", required=True, help="API or Service Account token for authentication")
def main(url, token):
    headers = {"Authorization": f"Bearer {token}", "Content-Type": "application/json"}
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
    main()
