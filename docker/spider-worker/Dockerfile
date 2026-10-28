FROM python:3.11-slim

ARG DOCKER_TAG=unknown
ARG IMAGE_NAME=unknown

ENV DEBIAN_FRONTEND=noninteractive \
    PYTHONUNBUFFERED=1 \
    PYTHONPATH=/opt/spider-worker:/opt/spider-worker/src \
    SPIDER_WORKDIR=/opt/spider-worker \
    AFFILIATION_DIR_LOCATION=/opt/spider-worker/.affiliation_dir.json \
    GOPATH=/root/go \
    PATH=$PATH:/root/go/bin \
    DOCKER_TAG=${DOCKER_TAG} \
    IMAGE_NAME=${IMAGE_NAME}

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        ca-certificates \
        libexpat1 \
        golang \
    && rm -rf /var/lib/apt/lists/* && \
    useradd -ms /bin/bash cmsjobmon

RUN go install github.com/nats-io/natscli/nats@v0.0.33 && \
    cp /root/go/bin/nats /usr/local/bin/nats

COPY requirements.txt /opt/spider-worker/requirements.txt
RUN pip install --no-cache-dir --upgrade pip setuptools wheel && \
    pip install --no-cache-dir -r /opt/spider-worker/requirements.txt
    
WORKDIR /opt/spider-worker

COPY src ./src

COPY main.py ./main.py

RUN chown -R cmsjobmon:cmsjobmon /opt/spider-worker

USER cmsjobmon

