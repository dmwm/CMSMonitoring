### CMS NATS clients
CMS [NATS](http://nats.io) is a cloud messaging service. Here you can find
basic Go-based clients for publisher and subscriber code. To compile the code
please following these steps:

```
# build executables
go build nats-sub.go # build NATS subscriber client
go build nats-pub.go # build NATS publisher client
```

### How to use NATS clients
To use NATS clients, e.g. nats-sub (subscriber), you need to have
CERN CAs certificates which can be obtained from
[ca.cern.ch](https://cafiles.cern.ch/cafiles/certificates/Download.aspx?ca=cern).
Then invoke nats-sub tool as following:
```
# get help message
nats-sub -h

# subscribe to cms exit code end-point
nats-sub -rootCAs "<path>/certs/CERN_CA.crt,<path>/certs/CERN_CA1.crt" "cms.wmarchive.exitCode.>"

# subscribe to cms exit code end-point and show time stamp
nats-sub -t -rootCAs "<path>/certs/CERN_CA.crt,<path>/certs/CERN_CA1.crt" "cms.wmarchive.exitCode.>"

# subscribe to cms exit code end-point, show time stamp and statistics
nats-sub -showStats -t -rootCAs "<path>/certs/CERN_CA.crt,<path>/certs/CERN_CA1.crt" "cms.wmarchive.exitCode.>"
```

Please note, NATS provides hierarchical structure of end-points. It starts from
the root `cms` followed by system, e.g. `wmarchive` or `dbs`, followed by
topic, e.g. `exitCode` or `site` or `task` or `campaign`, and finally followed
by attribute namd or value, e.g. `exitCode`. Each nested sub-topic is divided
by period `.`, such that we have `root.system.topic.value1.value2` convention.
The sites, datasets, tasks use a separator, in site case it is `_` and in case
of datasets, tasks it is `/`. These separatrs in NATS are replaced by periods
`.`. Therefore to refer to T3_US_Cornell site you'll need the following
end-point `cms.wmarchive.site.T3.US.Cornell`, if you want to listen to
all T3 sites you can use `>` placeholder on end-point, e.g.
`cms.wmarchive.site.T3.>`.

### How to start local NATS server
Sometimes we need to experiment/test our clients with local NATS server.
It is trivial to do. Please follow these steps:
```
# get nats server code
go get github.com/nats-io/nats-server

# create area with certificates
mkdir ~/certificates

# generate certificates for localhost
echo "### generate localhost-rootCA.key"
openssl genrsa -des3 -out localhost-rootCA.key 4096
echo "### generate localhost-rootCA.crt"
openssl req -x509 -new -nodes -key localhost-rootCA.key -sha256 -days 1024 -out localhost-rootCA.crt
echo "### generate localhost.key"
openssl genrsa -out localhost.key 2048
echo "### generate localhost.csr (Certificate Signing Request)"
openssl req -new -key localhost.key -out localhost.csr
echo "### verify localhost.csr"
openssl req -in localhost.csr -noout -text
echo "### generate localhost.crt"
openssl x509 -req -in localhost.csr -CA localhost-rootCA.crt -CAkey localhost-rootCA.key -CAcreateserial -out localhost.crt -days 500 -sha256

# start NATS server
nats-server --tls --tlscert ~/certificates/localhost.crt --tlskey ~/certificates/localhost.key --tlscacert ~/certificates/localhost-rootCA.crt --user <USER> --pass <PASSWORD> -DV
```
