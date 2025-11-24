FROM python:3.11-slim

ENV DEBIAN_FRONTEND=noninteractive \
    PYTHONUNBUFFERED=1

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        ca-certificates \
        curl \
    && rm -rf /var/lib/apt/lists/*

# Application user and directories
RUN useradd -ms /bin/bash cmsjobmon && \
    mkdir -p /opt/spider && \
    chown -R cmsjobmon:cmsjobmon /opt/spider

WORKDIR /opt/spider

COPY ./requirements.txt /tmp/requirements.txt
RUN pip install --no-cache-dir --upgrade pip && \
    pip install --no-cache-dir -r /tmp/requirements.txt

COPY ./src ./src
COPY ./*.sh ./bin/

RUN chown -R cmsjobmon:cmsjobmon /home/cmsjobmon /opt/spider

ENV PYTHONPATH=/opt/spider/src \
    SPIDER_WORKDIR=/opt/spider \
    AFFILIATION_DIR_LOCATION=/opt/spider/.affiliation_dir.json

USER cmsjobmon

# Default entrypoint simply invokes python. Override CMD/args per cronjob template.
ENTRYPOINT ["python"]
CMD ["--version"]

