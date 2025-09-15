import requests
import sys
import datetime
from typing import TypedDict, Literal, Union, Iterator, Any

from src.logging import logger


# TODO: Pass metrics as ConfigMaps to the K8s cronjobs
# Also can be extended to all other parameters
class RucioUsageFields(TypedDict):
    __name__: str
    aggregate: str
    country: str
    job: str
    rse: str
    rse_type: str
    rucioInstance: str
    source: str


class PromSeriesData(TypedDict):
    metric: RucioUsageFields
    values: list[list[int, str]]


class InnerData(TypedDict):
    result: list[PromSeriesData]


class RangeQueryResponse(TypedDict):
    status: Literal["success"]
    data: dict[str, Union[str, list[InnerData]]]


def query_promxy(promxy_url: str, params: dict[str, str]) -> RangeQueryResponse:
    logger.info(f"Querying Promxy: {promxy_url}")

    try:
        resp = requests.get(
            promxy_url,
            params=params,
        )
        resp.raise_for_status()
    except requests.RequestException as e:
        logger.error(f"Failed to fetch from Promxy: {e}")
        sys.exit(1)

    logger.debug(f"Final URL: {resp.url}")
    logger.debug(f"Promxy response status: {resp.status_code}")
    # logging.debug(f"Promxy response sample: {resp.json()['data']['result'][:5]}")

    return resp.json()


def generate_os_docs(responses: list[RangeQueryResponse], os_index: str) -> Iterator[dict[str:Any]]:
    for response in responses:
        for series in response["data"]["result"]:
            fields = series.get("metric", {})
            values = series.get("values", [])
            for ts, val in values:
                yield {
                    "_op_type": "index",
                    "_index": os_index,
                    "_source": {
                        "timestamp": datetime.datetime.fromtimestamp(
                            int(float(ts)), datetime.timezone.utc
                        ).isoformat()
                        + "Z",
                        "rse": fields.get("rse"),
                        "rse_type": fields.get("rse_type"),
                        "tier": fields.get("rse")[:2],
                        "country": fields.get("country"),
                        "rucioInstance": fields.get("rucioInstance"),
                        "source": fields.get("source"),
                        "used_bytes": float(val),
                    },
                }
