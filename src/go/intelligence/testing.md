### Testing intelligence module
We can easily test intelligence module by adding a fake/test alert
through `amtool` and checking if intelligence module will perform
all necessary steps to annotate the dashboards. Please adjust
and use the following command for testing:
```
#!/bin/bash
aurl="http://cms-monitoring.cern.ch:30093"
startAt="2020-10-30T19:05:00Z"
endAt="2020-10-30T19:15:00Z"
ssbNumber=OTG111112
stype="Planned Intervention"
severity="notification"
amtool alert add ssb-$ssbNumber tag=monitoring  \
    service=SSB severity=$severity type="$stype" \
    ssbNumber="$ssbNumber" \
    --annotation=shortDescription="TEST ALERT : Network Intervention in building 697" \
    --annotation=description="Network Services" \
    --annotation=type="$stype" \
    --annotation=severity="$severity" \
    --annotation=ssbNumber="$ssbNumber" \
    --start $startAt --end $endAt \
    --alertmanager.url $aurl
```

The start and end time stamp should be adjusted. The description shoudl contain
intervention to create appropriate annotations on cms monitoring Grafana dashboards.
