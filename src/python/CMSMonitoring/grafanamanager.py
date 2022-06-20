import os
import requests
import json
import logging
import re

logging.basicConfig(level=os.environ.get("LOGLEVEL", "DEBUG"))


class GrafanaManager:
    """An interface with the grafana API
    Attributes:
        GRAFANA_TOKEN
        GRAFANA_URL
        OUTPUT_FOLDER: local folder for methods which store dashoards.
    """

    def __init__(self, grafana_url=None, grafana_token=None, output_folder="./output"):
        assert (
            grafana_token and grafana_url
        ), "both grafana_token and grafana_url are required."
        self.GRAFANA_TOKEN = grafana_token
        self.GRAFANA_URL = (
            grafana_url[:-1] if grafana_url.endswith("/") else grafana_url
        )
        self.OUTPUT_FOLDER = output_folder

    def dashboard_duplicator(
        self,
        dashboard_uid=None,
        dashboards_query=None,
        datasources_replacements=None,
        title=None,
        store_only=False,
    ):
        """
            Create a copy of one or more dashboards,
            making datasources replacements if needed.
            (either uid or query must be specified).
            Args:
             - dashboard_uid uid of the original dashboard.
             - dashboards_query query to select the original dashboards
             - datasources_replacements list of pairs,
               each pair is composed by old and new datasource.
             - title new title, if uid is specified, or prefix for the titles.
             - store_only, do not publish to Grafana, just store localy the dashboards.
        """
        assert (
            dashboard_uid or dashboards_query
        ), "Either uid or query need to be specified"
        replace_title = False
        append_title = False
        if title:
            replace_title = True if dashboard_uid else False
            append_title = not replace_title
        dashboards = []
        if dashboards_query:
            dashboards = self.search_dasboards(
                query=dashboards_query, uid=dashboard_uid
            )
        if dashboard_uid:
            dashboards.append(self.get_dashboard(dashboard_uid))
        replacements = (
            dict(datasources_replacements) if datasources_replacements else {}
        )
        logging.debug("replacements {}", replacements)
        for d in dashboards:
            # Save a backup
            self.__save("backup", d)
            # Create the new dashboard
            new_dashboard = {
                "dashboard": d["dashboard"],
                "overwrite": False,
                "folderId": d["meta"]["folderId"],
            }
            new_dashboard["dashboard"]["id"] = None
            new_dashboard["dashboard"]["uid"] = None
            new_title = (
                title
                if replace_title
                else "{}_{}".format(new_dashboard["dashboard"]["title"], title)
                if append_title
                else new_dashboard["dashboard"]["title"]
            )
            new_dashboard["dashboard"]["title"] = new_title
            self.__replace_dataset(new_dashboard, replacements)
            self.__save("new", new_dashboard)
            if not store_only:
                logging.info("Try to create the dashboards in grafana")
                logging.info(new_title)
                logging.info(self.post_dashboard(new_dashboard))

    def text_template_replace(
        self,
        source_dashboard_query: dict,
        target_dashboards_query: dict,
        start_text: str = "<!--START_MENU-->",
        end_text: str = "<!--END_MENU-->",
    ) -> int:
        """
            Replaces the content between start_text and end_text
            of text panels in the matching
            target_dashboards for the one in the source dashboard.
            Returns the number of replaced dashboards.

        """
        _source_dash_list = self.search_dasboards(**source_dashboard_query)
        if not _source_dash_list:
            raise Exception("No dashboard matches the source query")
        _sd = _source_dash_list[0]
        _sp_content = None
        _search = f"(?sm){re.escape(start_text)}.*{re.escape(end_text)}"

        for panel in _sd["dashboard"]["panels"]:
            if start_text in panel.get("options").get("content"):

                _sp_content_match = re.search(_search, panel["options"]["content"])
                if _sp_content_match:
                    _sp_content = _sp_content_match[0]
                    break
        if not _sp_content:
            raise Exception("There is not matching content in the source dashboard")
        _target_dashboards = self.search_dasboards(**target_dashboards_query)
        to_update = []
        for _dash in _target_dashboards:
            for panel in _dash["dashboard"]["panels"]:
                # If dashboard json includes "options" which is parent of "content"
                if "options" in panel:
                    if start_text in panel.get("options").get("content", ""):
                        _tp_content = panel["options"]["content"]
                        _tp_content_replaced = re.sub(_search, _sp_content, _tp_content)
                        if _tp_content != _tp_content_replaced:
                            panel["options"]["content"] = _tp_content_replaced
                            _dash["message"] = "Updated using the grafana manager"
                            to_update.append(_dash)
                else:
                    if start_text in panel.get("content", ""):
                        _tp_content = panel["content"]
                        _tp_content_replaced = re.sub(_search, _sp_content, _tp_content)
                        if _tp_content != _tp_content_replaced:
                            panel["content"] = _tp_content_replaced
                            _dash["message"] = "Updated using the grafana manager"
                            to_update.append(_dash)
        for _dash in to_update:
            self.post_dashboard(_dash)
        return len(to_update)

    def search_dasboards(self, query=None, uid=None, tags=None):
        params = {}
        request_uri = "{uri}/api/search".format(uri=self.GRAFANA_URL)
        params["type"] = "dash_db"
        if query:
            params["query"] = query
        if tags:
            params["tag"] = tags
        dashboards_links = self.__make_request(request_uri, params=params)
        return [self.get_dashboard(x["uid"]) for x in dashboards_links]

    def get_dashboard(self, uid=None):
        request_uri = "{uri}/api/dashboards/uid/{uid}".format(
            uri=self.GRAFANA_URL, uid=uid
        )
        return self.__make_request(request_uri, params=None)

    def post_dashboard(self, dashboard):
        request_uri = "{uri}/api/dashboards/db".format(uri=self.GRAFANA_URL)
        return self.__make_request(request_uri, params=dashboard, type="post")

    def __make_request(self, uri, params, type="get"):
        headers = {"Authorization": "Bearer {}".format(self.GRAFANA_TOKEN)}
        verify = True
        # The monit-grafana-dev uses an invalid certificate
        if "-dev" in uri:
            verify = False
        response = None
        if type == "get":
            response = requests.get(uri, params=params, headers=headers, verify=verify)
        elif type == "post":
            response = requests.post(uri, json=params, headers=headers, verify=verify)
        if response:
            return json.loads(response.content)
        else:
            raise Exception("Something happen with the request, {}".format(response))

    def __save(self, subfolder, d):
        _folderId = (
            d["meta"]["folderId"] if "meta" in d else d.get("folderId", "unknown")
        )
        filename = os.path.join(
            self.OUTPUT_FOLDER,
            subfolder,
            "{folderId}_{title}.json".format(
                folderId=_folderId, title=d["dashboard"].get("title", "unknown")
            ),
        )
        os.makedirs(os.path.dirname(filename), exist_ok=True)
        with open(filename, "w") as f:
            json.dump(d, f, indent=4)

    def __replace_dataset(self, element, replacements):
        if isinstance(element, list):
            for e in element:
                self.__replace_dataset(e, replacements)
        elif isinstance(element, dict):
            for key in element:
                if key == "datasource":
                    element[key] = replacements.get(element[key], element[key])
                elif isinstance(element[key], (dict, list)):
                    self.__replace_dataset(element[key], replacements)
