"""
Various helper utilities for the HTCondor-ES integration
"""

import os
import pwd
import time
import shlex
import socket
import logging
import smtplib
import subprocess
import email.mime.text

import vals



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




def convert_dates_to_millisecs(record: dict) -> dict:
    for date_field in vals.date_vals:
        try:
            record[date_field] *= 1000
        except (KeyError, TypeError):
            continue

    return record