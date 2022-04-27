# Docker file for testing logstash easily
FROM docker.elastic.co/logstash/logstash:8.1.3
MAINTAINER Ceyhun Uzunoglu ceyhunuzngl@gmail.com

USER root
ENV TERM xterm-256color
ENV WDIR=/data
WORKDIR $WDIR

ADD server.py $WDIR
ADD filebeat.yaml $WDIR
ADD howtorun.txt $WDIR

RUN apt-get update && apt-get -y install apt-utils sudo python3 vim nano netcat jq locales locales-all && \
    rm -f /usr/bin/python && ln -s /usr/bin/python3 /usr/bin/python

# Install filebeat
RUN curl -L -O https://artifacts.elastic.co/downloads/beats/filebeat/filebeat-8.1.3-linux-x86_64.tar.gz	&& \
    tar xzvf filebeat-8.1.3-linux-x86_64.tar.gz	&& rm -rf filebeat-8.1.3-linux-x86_64.tar.gz &&	\
    mv filebeat-8.1.3-linux-x86_64 filebeat-8.1.3

# Get logstash.yml
RUN curl https://raw.githubusercontent.com/dmwm/CMSKubernetes/master/kubernetes/cmsweb/monitoring/logstash.yml > /usr/share/logstash/config/logstash.yml

# Get logstash.conf with replaced url
RUN curl https://raw.githubusercontent.com/dmwm/CMSKubernetes/master/kubernetes/cmsweb/monitoring/logstash.conf > /usr/share/logstash/pipeline/logstash.conf && \
    sed -i 's#http://monit-logs.cern.ch:10012/#http://localhost:10012#g' /usr/share/logstash/pipeline/logstash.conf

# Create directory for logs
RUN mkdir -p $WDIR/logs

ENV LC_ALL en_US.UTF-8
ENV LANG en_US.UTF-8
ENV LANGUAGE en_US.UTF-8

WORKDIR $WDIR
