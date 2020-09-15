### CMS NATS service
The [NATS](https://nats.io/) is a simple, secure and high performance open
source messaging system for cloud native applications, IoT messaging, and
microservices architectures.

In CMS we use it for quasi-real-time communication between services, e.g.
to monitor production workflow exit codes from WMArchive data.

It runs on dedicated cms-nats.cern.ch cluster and accessible both
inside and outside CERN network. There are two components:
- a nats publisher, a stream of data which are pushed to NATS, e.g. WMArchive
  service can send data about exit codes
- a nats subscribers, a set of clients which can consume data on certain topic,
  e.g. subscribe to WMArchive stream of data

### CMS NATS examples
Here we provide a few examples how to start using NATS and subscribe to
specific topic:
```
# subscribe to cms.wmarchive.exitCode end-point using
# provided credentials from /etc/nats/cms-auth file
# and using CERN ROOT CAs, the -t flag will turn on time stamp
/cvmfs/cms.cern.ch/cmsmon/nats-sub \
   -rootCAs=/etc/nats/CERN_CA.crt,/etc/nats/CERN_CA1.crt \
   -t -cmsAuth=/etc/nats/cms-auth \
   "cms.wmarchive.exitCode.>"
```

And, here are simplified examples of using different topics
```
# watch all exit codes
nats-sub “cms.wmarchive.exitCode.>”

# watch 8028 exit code only
nats-sub “cms.wmarchive.exitCode.8028”

# watch production on all T2 sites
nats-sub “cms.wmarchive.site.T2.>”

# watch production of specific campaign
nats-sub “cms.wmarchive.campaign.RunIISummer19UL17SIM”
```

The all set of options can be found through `-help` command, see
```
/cvmfs/cms.cern.ch/cmsmon/nats-sub -help
Usage: nats-sub [-s server] [-creds file] [-t] <subject>
  -attrs string
        comma separated list of attributes to show, e.g. campaign,dataset
  -cmsAuth string
        User cms auth file
  -creds string
        User NSC credentials
  -gatewayInstance string
        gateway instance name
  -gatewayJob string
        gateway job name
  -gatewayUri string
        URI of gateway server
  -h    Show help message
  -raw
        Show raw messages
  -rootCAs string
        Comma separated list of CERN Root CAs files
  -s string
        The nats server URLs (separated by comma) (default "cms-nats.cern.ch")
  -sep string
        message attribute separator, default 3 spaces (default "   ")
  -showStats
        Show stats instead of messages
  -statsBy string
        Show stats by given attribute
  -statsInterval int
        stats collect interval in seconds (default 10)
  -t    Display timestamps
  -userCert string
        x509 user certificate
  -userKey string
        x509 user key file
  -vmUri string
        VictoriaMetrics URI
```

And, similar for nats-pub tool
```
/cvmfs/cms.cern.ch/cmsmon/nats-pub -help
Usage: nats-pub [-s server] [-creds file] <subject> <msg>
  -cmsAuth string
        User cms auth file
  -creds string
        User NSC credentials
  -h    Show help message
  -rootCAs string
        Comma separated list of CERN Root CAs files
  -s string
        The nats server URLs (separated by comma) (default "cms-nats.cern.ch")
  -userCert string
        x509 user certificate
  -userKey string
        x509 user key file        
```
