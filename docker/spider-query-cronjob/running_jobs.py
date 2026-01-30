#!/usr/bin/env python
"""
Script for processing the contents of the CMS pool.
"""

import time
import signal
import os

from src.utils import get_schedds_from_file, global_logger
from src.query import query_running_jobs

# Ensure this script has a distinct OpenTelemetry service/trace name
os.environ.setdefault("SPIDER_TRACE_NAME", "spider_running_jobs")

from src.otel_setup import trace_span
import src.constants as const


@trace_span("running_jobs_main")
def main():
    starttime = time.time()
    global_logger.info("Starting spider_cms running jobs process.")
    signal.alarm(const.TIMEOUT_MINS * 60 + 60)

    # Get all the schedd ads (these are ClassAds; they can be sent directly
    # to worker processes, and `htcondor.Schedd` expects this type).
    schedd_ads = get_schedds_from_file(collectors_file=const.COLLECTORS_FILE)

    query_running_jobs(starttime, schedd_ads)

    return 0


if __name__ == "__main__":
    main()
