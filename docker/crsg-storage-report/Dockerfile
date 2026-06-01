FROM gitlab-registry.cern.ch/linuxsupport/alma9-base

ENV PYTHONUNBUFFERED=1 \
    PYTHONDONTWRITEBYTECODE=1 \
    PATH=/usr/local/bin:$PATH \
    PYTHONPATH=/app:$PYTHONPATH

WORKDIR /app

RUN dnf -y install epel-release
RUN dnf -y install python3-opensearch-py+kerberos
RUN dnf -y install python3-requests-gssapi

COPY src/ ./src
COPY promxy_to_opensearch.py .
COPY entrypoint.sh .

CMD ["/app/entrypoint.sh"]

