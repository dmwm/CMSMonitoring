#!/usr/bin/env python
"""
Script for processing the contents of the CMS pool.
"""

import time
import os

# Ensure this script has a distinct OpenTelemetry service name (for tracing)
# Must be set BEFORE any src.* imports that may load constants.py
os.environ.setdefault("OTEL_SERVICE_NAME", "spider-job-history")

from src.utils import get_schedds_from_file, global_logger
from src.history import query_job_history
from src.otel_setup import trace_span
import src.constants as const
from opentelemetry import trace


@trace_span("job_history_main")
def main():
    starttime = time.time()
    global_logger.info("Starting spider_cms history process.")

    # Get all the schedd ads (these are ClassAds; they can be sent directly
    # to worker processes, and `htcondor.Schedd` expects this type).
    schedd_ads = get_schedds_from_file(collectors_file=const.COLLECTORS_FILE)

    counts = query_job_history(schedd_ads, starttime)
    trace.get_current_span().set_attribute("job.count", counts["count"])
    trace.get_current_span().set_attribute("job.published_count", counts["published_count"])
    return 0


if __name__ == "__main__":
    main()
