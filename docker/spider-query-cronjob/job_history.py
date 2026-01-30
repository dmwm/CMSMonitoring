#!/usr/bin/env python
"""
Script for processing the contents of the CMS pool.
"""

import time
import signal

from src.utils import get_schedds_from_file, global_logger
from src.history import query_job_history
from src.otel_setup import trace_span
import src.constants as const


@trace_span("job_history_main")
def main():
    starttime = time.time()
    global_logger.info("Starting spider_cms history process.")
    signal.alarm(const.TIMEOUT_MINS * 60 + 60)

    # Get all the schedd ads (these are ClassAds; they can be sent directly
    # to worker processes, and `htcondor.Schedd` expects this type).
    schedd_ads = get_schedds_from_file(collectors_file=const.COLLECTORS_FILE)

    query_job_history(schedd_ads, starttime)

    return 0


if __name__ == "__main__":
    main()
