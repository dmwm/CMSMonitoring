import os
import logging
from datetime import datetime, timezone, timedelta


DEBUG_MODE = os.getenv("DEBUG_MODE", "false").lower() == "true"
LOG_LEVEL = logging.DEBUG if DEBUG_MODE else logging.INFO

DRY_RUN = os.getenv("DRY_RUN", "false").lower() == "true"

PROMXY_URL = os.environ.get(
    "PROMXY_URL", "http://cms-monitoring.cern.ch:30082/api/v1/query_range"
)
PROM_QUERY = os.environ.get("PROM_QUERY", "avg_over_time:rucio_report_used_space:1h")
INDEX = os.environ.get("OPENSEARCH_INDEX", "rucio-used-space-1h")

OS_HOST = os.environ.get("OPENSEARCH_HOST", "https://os-cms.cern.ch:443/os")
CERT_PATH = os.environ.get("CERT_PATH", "/etc/pki/tls/certs/ca-bundle.trust.crt")

STEP = os.environ.get("STEP", "3600")
START_DATE = os.environ.get("START_DATE", None)
END_DATE = os.environ.get("END_DATE", None)

START = (
    datetime.strptime(START_DATE, "%Y-%m-%d")
    if START_DATE and END_DATE
    else datetime.now(timezone.utc) - timedelta(days=30)
)
END = (
    datetime.strptime(END_DATE, "%Y-%m-%d")
    if START_DATE and END_DATE
    else datetime.now(timezone.utc)
)
