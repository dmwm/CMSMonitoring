import datetime
from typing import Any, Iterable, Optional
from opensearchpy import OpenSearch, helpers, RequestsHttpConnection
import constants as const
import src.vals as vals


_INDEX_SETTINGS = {"mapping.total_fields.limit": 2000}
_DATE_MAPPING = {"type": "date", "format": "epoch_millis"}
_OS_CLIENT: Optional[OpenSearch] = None
_ENSURED_DAILY_INDICES: set[str] = set()

def get_opensearch_client() -> OpenSearch:
    global _OS_CLIENT
    if _OS_CLIENT is None:
        _OS_CLIENT = OpenSearch(
            hosts=[const.OS_HOST],
            http_auth=(const.OS_USERNAME, const.OS_PASSWORD),
            use_ssl=True,
            verify_certs=True,
            ca_certs=const.OS_CERT_PATH,
            connection_class=RequestsHttpConnection,
        )
    return _OS_CLIENT


def get_daily_index_name(timestamp: float, index_prefix: str) -> str:
    """
    Generate a daily index name from a timestamp and prefix.

    Args:
        timestamp: Unix timestamp (seconds since epoch)
        index_prefix: Prefix for the index name (e.g., 'cms', 'monitoring')

    Returns:
        Index name in format: '{prefix}-YYYY-MM-DD'

    Example:
        get_daily_index(1704067200.0, 'cms') -> 'cms-2024-01-01'
    """
    dt = datetime.datetime.fromtimestamp(timestamp, datetime.timezone.utc)
    return f"{index_prefix}-{dt.strftime('%Y-%m-%d')}"


def ensure_daily_index(
    os_client: OpenSearch,
    index_name: str,
) -> bool:
    """
    Ensure a daily index exists, creating it if necessary.

    Args:
        os_client: OpenSearch client instance
        index_name: Name of the index to ensure exists
        mappings: Optional index mappings to apply
        settings: Optional index settings to apply

    Returns:
        True if index was created, False if it already existed
    """
    if index_name in _ENSURED_DAILY_INDICES:
        return False

    if os_client.indices.exists(index=index_name):
        _ENSURED_DAILY_INDICES.add(index_name)
        return False

    os_client.indices.create(index=index_name, body=_build_index_body())
    _ENSURED_DAILY_INDICES.add(index_name)
    return True


def _build_index_body() -> dict[str, Any]:
    return {
        "settings": {"index": _INDEX_SETTINGS},
        "mappings": {
            "properties": {field: dict(_DATE_MAPPING) for field in vals.date_vals}
        },
    }


def os_upload_docs_in_bulk(
    doc_iterable: Iterable,
    index_prefix: Optional[str] = None,
    timestamp: Optional[float] = None,
):
    """
    Upload documents in bulk to OpenSearch.

    Args:
        os_host: OpenSearch host URL
        ca_cert_path: Path to CA certificate file
        credentials: Tuple of (username, password)
        doc_iterable: Iterable of documents to upload
        timestamp: Optional Unix timestamp. It will create a daily
            index by appending it's date to the index_prefix.
        index_prefix: Prefix for daily index. It will be used
            with the timestamp to create a daily index.

    Returns:
        Tuple of (success_count, failures_list)
    """
    os_client = get_opensearch_client()
    index = get_daily_index_name(
        timestamp=timestamp,
        index_prefix=index_prefix,
    )
    ensure_daily_index(os_client, index)
    success, failures = helpers.bulk(
        os_client, doc_iterable, max_retries=3, index=index
    )
    return success, failures
