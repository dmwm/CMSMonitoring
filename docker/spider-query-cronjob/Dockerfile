FROM python:3.11-slim

ENV DEBIAN_FRONTEND=noninteractive \
    PYTHONUNBUFFERED=1

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        ca-certificates \
        curl \
        libexpat1 \
    && rm -rf /var/lib/apt/lists/*

RUN useradd -ms /bin/bash cmsjobmon && \
    mkdir -p /opt/spider/ && \
    chown -R cmsjobmon:cmsjobmon /opt/spider

WORKDIR /opt/spider

COPY ./requirements.txt /tmp/requirements.txt
RUN pip install --no-cache-dir --upgrade pip && \
    pip install --no-cache-dir -r /tmp/requirements.txt

COPY ./src ./src
COPY ./running_jobs.py ./running_jobs.py
COPY ./job_history.py ./job_history.py
COPY ./collectors.json ./etc/collectors.json
# For testing with only specific collectors instead (lower volume of data)
# COPY ./test_collectors.json ./etc/collectors.json
RUN chmod +x ./*.py

RUN chown -R cmsjobmon:cmsjobmon /home/cmsjobmon /opt/spider

# Build arguments for docker image (used for monitoring)
ARG DOCKER_TAG=unknown
ARG IMAGE_NAME=unknown

ENV PYTHONPATH=/opt/spider/src:$PYTHONPATH \
    SPIDER_WORKDIR=/opt/spider \
    AFFILIATION_DIR_LOCATION=/opt/spider/.affiliation_dir.json \
    DOCKER_TAG=${DOCKER_TAG} \
    IMAGE_NAME=${IMAGE_NAME} \
    COLLECTORS_FILE=/opt/spider/etc/collectors.json

USER cmsjobmon
