FROM golang:latest
LABEL maintainer="Rahul Indra indrarahul2013@gmail.com"

WORKDIR /monit

# INPUT REQUIRED
#Copy the intelligence module configuration file. Here you need to pass the absolute path of the config file.
COPY <INT_CONFIG_FILE_PATH> .

#Test Wrapper script basically runs AlertManager in the container in background and then run the test for intelligence module.
COPY test_wrapper.sh .

#Giving Executable permission to test_wrapper.sh
RUN chmod +x test_wrapper.sh

#Official Downloading of alertmanager for testing purpose. 
# ---> Current release v0.21.0
# You need to change the version number only (in future), rest processes are taken care of.
RUN wget "https://github.com/prometheus/alertmanager/releases/download/v0.21.0/alertmanager-0.21.0.linux-amd64.tar.gz" -O am.tar.gz
RUN tar -xzf am.tar.gz --one-top-level=am --strip-components 1

# Cloning the latest version of CMSMonitoring Repository
RUN git clone https://github.com/dmwm/CMSMonitoring.git
# Building the test script for intelligence module
RUN cd CMSMonitoring && export GOPATH=`pwd` && go install go/intelligence/test

# Entrypoint of the test
CMD ./test_wrapper.sh