"""
Various helper utilities for the HTCondor-ES integration
"""

import os
import pwd
import sys
import time
import shlex
import socket
import logging
import smtplib
import subprocess
import email.mime.text
import json

import classad
import htcondor


def get_schedds_from_file(collectors_file: str = None, schedd_filter: str = None):
    schedds = []
    names = set[str]()
    try:
        # TODO: Check type validation for pools
        with open(collectors_file, "r") as f:
            pools: dict[str, list[str]] = json.load(f)
        for pool in pools.keys():
            _pool_schedds = get_schedds(collectors=pools[pool], pool_name=pool, schedd_filter=schedd_filter)
            schedds.extend([s for s in _pool_schedds if s.get("Name") not in names])
            names.update([s.get("Name") for s in _pool_schedds])

    except (IOError, json.JSONDecodeError):
        schedds = get_schedds(schedd_filter=schedd_filter)

    global_logger.info("&&& There are %d schedds to query.", len(schedds))
    return schedds


def get_schedds(collectors: list[str] = None, pool_name: str = "Unknown", schedd_filter: str = None) -> list[dict]:
    """
    Return a list of schedd ads representing all the schedds in the pool.
    """
    collectors = collectors or []
    schedd_query = classad.ExprTree("!isUndefined(CMSGWMS_Type)")

    schedd_ads: dict[str, dict] = {}
    for host in collectors:
        coll = htcondor.Collector(host)
        try:
            schedds = coll.query(
                htcondor.AdTypes.Schedd,
                schedd_query,
                projection=["MyAddress", "ScheddIpAddr", "Name", "TotalRunningJobs", "TotalIdleJobs", "TotalHeldJobs"],
            )
        except IOError as e:
            logging.warning(str(e))
            continue

        for schedd in schedds:
            try:
                schedd["CMS_Pool"] = pool_name
                schedd_ads[schedd["Name"]] = schedd
                schedd_ads[schedd["Name"]]["TotalJobs"] = schedd["TotalRunningJobs"] + schedd["TotalIdleJobs"] + schedd["TotalHeldJobs"]
            except KeyError:
                pass

    schedd_ads = list(schedd_ads.values())
    schedd_ads.sort(key=lambda x: x["TotalJobs"], reverse=True)

    if schedd_filter:
        return [s for s in schedd_ads if s["Name"] in schedd_filter.split(",")]
    
    return schedd_ads


def send_email_alert(recipients, subject, message):
    """
    Send a simple email alert (typically of failure).
    """
    # TMP: somehow send_email_alert still sending alerts
    if not recipients:
        return
    msg = email.mime.text.MIMEText(message)
    msg["Subject"] = "%s - %sh: %s" % (
        socket.gethostname(),
        time.strftime("%b %d, %H:%M"),
        subject,
    )

    domain = socket.getfqdn()
    uid = os.geteuid()
    pw_info = pwd.getpwuid(uid)
    if "cern.ch" not in domain:
        domain = "%s.unl.edu" % socket.gethostname()
    msg["From"] = "%s@%s" % (pw_info.pw_name, domain)
    msg["To"] = recipients[0]

    try:
        sess = smtplib.SMTP("localhost")
        sess.sendmail(msg["From"], recipients, msg.as_string())
        sess.quit()
    except Exception as exn:  # pylint: disable=broad-except
        logging.warning("Email notification failed: %s", str(exn))


def time_remaining(starttime: float, timeout: int = 3600, positive: bool = True) -> int:
    """
    Return the remaining time (in seconds) until starttime + timeout
    Returns 0 if there is no time remaining
    """
    elapsed = time.time() - starttime
    if positive:
        return max(0, timeout - elapsed)
    return timeout - elapsed


def set_up_logging(log_level: int = logging.INFO) -> logging.Logger:
    """Configure root logger with rotating file handler"""
    logger = logging.getLogger()
    logger.setLevel(log_level)

    if log_level <= logging.INFO:
        logging.getLogger("CMSMonitoring.StompAMQ").setLevel(log_level + 10)
        logging.getLogger("stomp.py").setLevel(log_level + 10)

    # Check if a StreamHandler already exists to avoid duplicate handlers
    # This prevents duplicate log messages when the function is called multiple times
    has_stream_handler = any(
        isinstance(handler, logging.StreamHandler) and handler.stream == sys.stdout
        for handler in logger.handlers
    )
    
    if not has_stream_handler:
        stream_handler = logging.StreamHandler(sys.stdout)
        stream_handler.setFormatter(logging.Formatter('%(asctime)s : %(name)s:%(levelname)s PID: %(process)d - %(message)s'))
        logger.addHandler(stream_handler)
    
    return logger

def collect_metadata():
    """
    Return a dictionary with:
    - hostname
    - username
    - current time (in epoch millisec)
    - hash of current git commit
    """
    result = {}
    result["spider_git_hash"] = get_githash()
    result["spider_hostname"] = socket.gethostname()
    result["spider_username"] = pwd.getpwuid(os.geteuid()).pw_name
    result["spider_runtime"] = int(time.time() * 1000)
    return result


def get_githash():
    """Returns the git hash of the current commit in the scripts repository"""
    gitwd = os.path.dirname(os.path.realpath(__file__))
    cmd = r"git rev-parse --verify HEAD"
    try:
        call = subprocess.Popen(shlex.split(cmd), stdout=subprocess.PIPE, cwd=gitwd)
        out, err = call.communicate()
        return str(out.strip())

    except Exception as e:
        logging.warning(str(e))
        return "unknown"


global_logger = set_up_logging()