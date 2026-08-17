#!/usr/bin/env python
# -*- coding: utf-8 -*-
# Author: Christian Ariza <christian.ariza AT gmail [DOT] com>
# This script fetches affiliation data from CRIC API and publishes it to JetStream KV store
# (an indexed structure of username/login and affiliation institution and country from cric data)
# How to run:
#       python affiliation_cache.py
# This will fetch the latest data from CRIC API and update the cache in JetStream KV store.
# This script can be setup as a daily cronjob to keep the cache up to date.
import traceback
import asyncio
import json
import logging
import os

import requests
from nats.aio.client import Client as NATS

AFFILIATION_API_URL = os.getenv("AFFILIATION_API_URL", "https://cms-cric.cern.ch/api/accounts/user/query/?json")
CA_CERT = os.getenv("CA_CERT", "/etc/pki/tls/certs/CERN-bundle.pem")
ROBOT_CERT = os.getenv("ROBOT_CERT", "/etc/secrets/robot/cert/robotcert.pem")
ROBOT_KEY = os.getenv("ROBOT_KEY", "/etc/secrets/robot/key/robotkey.pem")
NATS_SERVER = os.getenv("NATS_SERVER", "nats://nats.cluster.local:4222")
KV_BUCKET = os.getenv("KV_BUCKET", "spider_affiliations")
KV_KEY = os.getenv("KV_KEY", "affiliations")


def fetch_affiliations(
    service_url=AFFILIATION_API_URL,
    robot_cert=ROBOT_CERT,
    robot_key=ROBOT_KEY,
    ca_cert=CA_CERT,
):
    """
    Fetch affiliation data from CRIC API.
    Returns an inverted index of institutions by person login. e.g.:

    {
        'valya':{u'country': u'US',
                u'institute': u'Cornell University'},
        'belforte': {u'country': u'IT',
                     u'institute': u'Universita e INFN Trieste'}
        ...
    }
    """
    logging.info("Fetching affiliation data from CRIC API")
    cert = (robot_cert, robot_key)
    response = requests.get(service_url, cert=cert, verify=ca_cert)
    response.raise_for_status()

    _json = json.loads(response.text)
    affiliations_dict = {}
    for person in list(_json.values()):
        login = None
        for profile in person["profiles"]:
            if "login" in profile:
                login = profile["login"]
                break
        if login and "institute" in person:
            affiliations_dict[login] = {
                "institute": person["institute"],
                "country": person["institute_country"],
                "dn": person["dn"],
            }

    logging.debug("Fetched %d affiliations from CRIC API", len(affiliations_dict))
    return affiliations_dict


def publish_to_kv(
    affiliations_dict,
    nats_server=NATS_SERVER,
    kv_bucket_name=KV_BUCKET,
):
    """
    Publish affiliations to JetStream KV store.
    """
    nats_connection = None
    try:
        # Create event loop
        try:
            loop = asyncio.get_event_loop()
            if loop.is_closed():
                raise RuntimeError("Event loop is closed")
        except RuntimeError:
            loop = asyncio.new_event_loop()
            asyncio.set_event_loop(loop)

        # Connect to NATS
        nats_connection = NATS()
        nats_servers = [s.strip() for s in nats_server.split(",")]
        loop.run_until_complete(nats_connection.connect(servers=nats_servers))
        jetstream = nats_connection.jetstream()
        logging.debug("Connected to NATS JetStream at %s", nats_servers)

        # Get KV store
        kv = loop.run_until_complete(jetstream.key_value(kv_bucket_name))

        # Publish to KV store
        value = json.dumps(affiliations_dict).encode("utf-8")
        loop.run_until_complete(kv.put(KV_KEY, value))
        logging.info(
            "Successfully published %d affiliations to KV store %s",
            len(affiliations_dict),
            kv_bucket_name,
        )
    except Exception as e:
        logging.error("Failed to publish affiliations to KV store: %s", str(e))
        raise
    finally:
        # Close NATS connection
        if nats_connection is not None and not nats_connection.is_closed:
            try:
                loop.run_until_complete(nats_connection.close())
            except Exception as e:
                logging.debug("Error closing NATS connection: %s", str(e))


def fetch_and_publish(
    service_url=AFFILIATION_API_URL,
    robot_cert=ROBOT_CERT,
    robot_key=ROBOT_KEY,
    ca_cert=CA_CERT,
    nats_server=NATS_SERVER,
    kv_bucket_name=KV_BUCKET,
):
    """
    Fetch affiliation data from CRIC API and publish to JetStream KV store.
    """
    affiliations_dict = fetch_affiliations(
        service_url=service_url, robot_cert=robot_cert, robot_key=robot_key, ca_cert=ca_cert
    )

    if affiliations_dict:
        publish_to_kv(
            affiliations_dict, nats_server=nats_server, kv_bucket_name=kv_bucket_name
        )
    else:
        logging.warning("No affiliations to publish")
        

def main():
    """
    Fetch affiliation data from CRIC API and update the cache in JetStream KV store.
    """
    try:
        fetch_and_publish()
    except Exception as e:
        traceback.print_exc()
        raise


if __name__ == "__main__":
    main()