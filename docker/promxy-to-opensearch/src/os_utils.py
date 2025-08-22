from typing import Iterable
from opensearchpy import OpenSearch, helpers, RequestsHttpConnection
from requests_gssapi import HTTPSPNEGOAuth, OPTIONAL

def get_opensearch_client(os_host: str, ca_cert_path: str) -> int:
    return OpenSearch(
        [os_host],
        use_ssl=True,
        verify_certs=True,
        ca_certs=ca_cert_path,
        connection_class=RequestsHttpConnection,
        http_auth=HTTPSPNEGOAuth(mutual_authentication=OPTIONAL),
    )


def os_upload_docs_in_bulk(os_host: str, ca_cert_path: str, doc_iterable: Iterable):
    os_client = get_opensearch_client(os_host=os_host, ca_cert_path=ca_cert_path)

    success, failures = helpers.bulk(os_client, doc_iterable, max_retries=3)

    return success, failures
