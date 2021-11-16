# Docker file for testing logstash easily
FROM centos
MAINTAINER Ceyhun Uzunoglu ceyhunuzngl@gmail.com

ENV TERM xterm-256color

ENV WDIR=/data
RUN mkdir -p $WDIR

ADD logstash.repo /etc/yum.repos.d/logstash.repo
ADD bashrc_extension $WDIR
ADD server.py $WDIR

RUN yum -y update && \
    rpm --import https://artifacts.elastic.co/GPG-KEY-elasticsearch && \
    yum install -y glibc-langpack-en perl sudo nano which nc jq logstash python3 vim && \
    echo 'source ${WDIR}/bashrc_extension' >> ~/.bashrc && \
    source ~/.bashrc

WORKDIR $WDIR
