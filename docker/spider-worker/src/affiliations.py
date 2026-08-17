#!/usr/bin/env python
# -*- coding: utf-8 -*-
# Author: Christian Ariza <christian.ariza AT gmail [DOT] com>
# pylint: disable=line-too-long
import asyncio
import json
import logging
from nats.aio.client import Client as NATS

import constants as const

_KV_KEY = "affiliations"

logger = logging.getLogger(__name__)


class AffiliationException(Exception):
    """
    Exception wrapper for problems that prevents us to obtain the affiliation info.
    """
    pass


def get_affiliations_dict(nats_server=const.NATS_SERVER, kv_bucket_name=const.AFFILIATION_KV_BUCKET):
    """
    Load affiliations dict from JetStream KV store.
    
    Returns a dictionary mapping login to affiliation info, e.g.:
    {
        'valya': {'country': 'US', 'institute': 'Cornell University', 'dn': '...'},
        'belforte': {'country': 'IT', 'institute': 'Universita e INFN Trieste', 'dn': '...'},
        ...
    }
    
    Raises AffiliationManagerException if cache not found or cannot be loaded.
    """
    try:
        # Get or create event loop
        try:
            loop = asyncio.get_event_loop()
            if loop.is_closed():
                raise RuntimeError("Event loop is closed")
        except RuntimeError:
            loop = asyncio.new_event_loop()
            asyncio.set_event_loop(loop)
        
        # Connect to NATS
        nats_connection = NATS()
        loop.run_until_complete(nats_connection.connect(servers=nats_server))
        jetstream = nats_connection.jetstream()
        
        # Get KV store
        kv = loop.run_until_complete(jetstream.key_value(kv_bucket_name))
        
        # Get affiliations entry
        entry = loop.run_until_complete(kv.get(_KV_KEY))
        if entry is None:
            loop.run_until_complete(nats_connection.close())
            raise AffiliationException(
                f"Affiliation cache not found in KV store {kv_bucket_name}. "
                "Ensure the affiliation-cache cronjob has populated the cache."
            )
        
        # Parse JSON
        affiliations = json.loads(entry.value.decode("utf-8"))
        
        # Close connection
        loop.run_until_complete(nats_connection.close())
        
        logger.info("Loaded %d affiliations from KV store %s", len(affiliations), kv_bucket_name)
        return affiliations
        
    except Exception as e:
        logger.error("Failed to load affiliations from KV store: %s", str(e))
        raise AffiliationException(f"Failed to load affiliations: {str(e)}") from e


def get_affiliation_by_dn(affiliations_dict, dn):
    """
    Get affiliation info by DN (Distinguished Name).
    
    Args:
        affiliations_dict: The affiliations dictionary from get_affiliations_dict()
        dn: User DN (Distinguished Name)
        
    Returns:
        Dictionary with 'institute' and 'country' keys, or None if not found.
    """
    for affiliation in affiliations_dict.values():
        if affiliation.get("dn") == dn:
            return affiliation
    return None
