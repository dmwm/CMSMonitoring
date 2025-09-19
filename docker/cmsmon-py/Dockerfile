# Pull python:3.9-slim through CERN's pull-through cache for DockerHub
FROM registry.cern.ch/docker.io/library/python:3.9-slim

ENV WDIR=/data
ENV PYTHONPATH="${PYTHONPATH}:${WDIR}"
ENV LC_ALL=C.UTF-8 LANG=C.UTF-8

WORKDIR $WDIR

RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    krb5-user \
    && rm -rf /var/lib/apt/lists/*

RUN pip install --no-cache-dir click pandas rucio-clients schema stomp.py==7.0.0

COPY src/ ./
