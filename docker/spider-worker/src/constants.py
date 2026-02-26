import os
import uuid

# IMAGE_NAME has to start with `spider-`, otherwise the Opensearch logs index
# template won't get the data properly.
IMAGE_NAME = os.environ.get("IMAGE_NAME")
EXECUTION_ID = str(uuid.uuid4())

TIMEOUT_MINS = os.environ.get("TIMEOUT_MINS", 60)
WORKDIR = os.getenv("SPIDER_WORKDIR", "/opt/spider-worker")
SECRET_DIR = os.environ.get("SECRET_DIR", "/etc/secrets")
DOCKER_TAG = os.environ.get("DOCKER_TAG", "unknown")
DRY_RUN = os.getenv("DRY_RUN", "false").lower() == "true"


# We reduce the data by default to save space in the Opensearch index.
REDUCE_DATA = os.getenv("REDUCE_DATA", "true").lower() == "true"

# NATS
NATS_SERVER = os.environ.get("NATS_SERVER", "nats://nats.cluster.local:4222")
NATS_STREAM_NAME = os.environ.get("NATS_STREAM_NAME", "CMS_HTCONDOR_QUEUE")
NATS_SUBJECT = os.environ.get("NATS_SUBJECT", "cms.htcondor.queue.job")
NATS_CONSUMER_NAME = os.environ.get("NATS_CONSUMER_NAME", "spider-worker")

# Affiliation KV bucket
AFFILIATION_KV_BUCKET = os.environ.get("AFFILIATION_KV_BUCKET", "spider_affiliations")

# OpenSearch
OS_HOST = os.environ.get("OS_HOST", "https://os-cms.cern.ch:443/os")
OS_INDEX_TEMPLATE = os.environ.get("OS_INDEX_TEMPLATE", "cms-htcondor-%Y-%m-%d")
OS_CERT_PATH = os.environ.get("OS_CERT_PATH", "/etc/pki/tls/certs/ca-bundle.pem")
OS_USERNAME = os.environ.get("OS_USERNAME")
OS_PASSWORD = os.environ.get("OS_PASSWORD")

# AMQ
AMQ_USERNAME = os.environ.get("AMQ_USERNAME")
AMQ_PASSWORD = os.environ.get("AMQ_PASSWORD")
AMQ_PRODUCER = os.environ.get("AMQ_PRODUCER") # For testing, set to "condor-test"
AMQ_TOPIC = os.environ.get("AMQ_TOPIC") # For testing, set to  "/topic/cms.jobmon.condor"
AMQ_BROKER = os.environ.get("AMQ_BROKER") # For testing, set to "cms-test-mb.cern.ch"
AMQ_PORT = os.environ.get("AMQ_PORT", 61313)
SPIDER_VALIDATION_SCHEMA = os.environ.get("SPIDER_VALIDATION_SCHEMA", "/etc/secrets/spider_validation_schema.json")

# OpenTelemetry
OTEL_ENDPOINT = os.environ.get("OTEL_ENDPOINT", "opentelemetry-collector.opentelemetry.svc.cluster.local:4317")
OTEL_METRIC_EXPORT_INTERVAL = os.environ.get("OTEL_METRIC_EXPORT_INTERVAL", "15000")
OTEL_SERVICE_NAME = os.environ.get("OTEL_SERVICE_NAME", "spider-worker")
OTEL_USERNAME = os.environ.get("OPENTELEMETRY_USERNAME", "")
OTEL_PASSWORD = os.environ.get("OPENTELEMETRY_PASSWORD", "")
