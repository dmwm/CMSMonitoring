FROM python:3.11-slim

ENV PYTHONUNBUFFERED=1 \
    PYTHONPATH=/opt/spider

WORKDIR /opt/spider

RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates && \
    rm -rf /var/lib/apt/lists/* && \
    useradd -m -s /bin/bash cmsjobmon && \
    pip install --no-cache-dir --upgrade pip

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt && \
    chown -R cmsjobmon:cmsjobmon /opt/spider

COPY affiliation_cache.py ./

USER cmsjobmon

CMD ["python", "affiliation_cache.py"]
