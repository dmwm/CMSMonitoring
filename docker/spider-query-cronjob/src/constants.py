import os

# IMAGE_NAME has to start with `spider-`, otherwise the Opensearch index
# template won't get the data properly.
IMAGE_NAME = os.getenv("IMAGE_NAME", "spider-query-cronjob")
DOCKER_TAG = os.getenv("DOCKER_TAG", "unknown")

DRY_RUN = os.getenv("DRY_RUN", "false").lower() == "true"
MAX_DOCUMENTS = os.getenv("MAX_DOCUMENTS", 0)
MAX_PROCESSES = os.getenv("MAX_PROCESSES", 16)
MAX_HISTORY_PROCESSES = int(os.getenv("MAX_HISTORY_PROCESSES", 10))


FEED_ES = os.getenv("FEED_ES", "true").lower() == "true"
FEED_AMQ = os.getenv("FEED_AMQ", "true").lower() == "true"
# TODO: Look into the alerts
EMAIL_ALERTS = os.getenv("EMAIL_ALERTS", "").split(",") # "cms-comp-monit-alerts@cern.ch"

ROBOT_CERT = os.getenv("ROBOT_CERT", "/etc/secrets/robot/cert/robotcert.pem")
ROBOT_KEY = os.getenv("ROBOT_KEY", "/etc/secrets/robot/key/robotkey.pem")
CA_CERT = os.getenv("CA_CERT", "/etc/pki/tls/certs/CERN-bundle.pem")
AFFILIATION_URL = os.getenv("AFFILIATION_URL", "https://cms-cric.cern.ch/api/accounts/user/query/?json")
WORKDIR = os.getenv("SPIDER_WORKDIR", "/opt/spider")
AFFILIATION_DIR_LOCATION = os.getenv("AFFILIATION_DIR_LOCATION", "/opt/spider/.affiliation_dir.json")
COLLECTORS_FILE = os.getenv("COLLECTORS_FILE", "/opt/spider/etc/collectors.json")
REDUCE_DATA = os.getenv("REDUCE_DATA", "false").lower() == "true"
# NATS
NATS_SERVER = os.getenv("NATS_SERVER", "nats://nats.cluster.local:4222")
NATS_STREAM_NAME = os.getenv("NATS_STREAM_NAME", "CMS_HTCONDOR_QUEUE")
NATS_SUBJECT = os.getenv("NATS_SUBJECT", "cms.htcondor.queue.job")
CHECKPOINT_KV_BUCKET = os.getenv("CHECKPOINT_KV_BUCKET", "spider_checkpoints")
NATS_PUBLISH_TIMEOUT = int(os.getenv("NATS_PUBLISH_TIMEOUT", 10))  # seconds
NATS_BATCH_SIZE = int(os.getenv("NATS_BATCH_SIZE", 100))

# OpenTelemetry
OTEL_ENDPOINT = os.getenv("OTEL_EXPORTER_OTLP_ENDPOINT", "opentelemetry-collector.opentelemetry.svc.cluster.local:4317")
OTEL_METRIC_EXPORT_INTERVAL = os.getenv("OTEL_METRIC_EXPORT_INTERVAL", "15000")
OTEL_SERVICE_NAME = os.getenv("OTEL_SERVICE_NAME", "spider-query-cronjob")
OTEL_USERNAME = os.getenv("OPENTELEMETRY_USERNAME")
OTEL_PASSWORD = os.getenv("OPENTELEMETRY_PASSWORD")

TIMEOUT_MINS = os.getenv("TIMEOUT_MINS", 60)

# History
# If last query time in checkpoint is too old, but not from crab, results older
# than "now()-RETENTION_POLICY" will be ignored.
RETENTION_POLICY = os.getenv("RETENTION_POLICY", 3600)  # 1 hour
# Even in checkpoint.json last query time is older than this,
# results older than "now()-CRAB_MAX_QUERY_TIME_SPAN" will be ignored.
CRAB_MAX_QUERY_TIME_SPAN = os.getenv("CRAB_MAX_QUERY_TIME_SPAN", 12 * 3600)  # 12 hours
# Main query time, should be same with cron schedule.
QUERY_TIME_PERIOD = os.getenv("QUERY_TIME_PERIOD", 720)  # 12 minutes
# Maximum number of minutes to query for history.
HISTORY_QUERY_MAX_N_MINUTES = os.getenv("HISTORY_QUERY_MAX_N_MINUTES", 60)  # 1 hour
