#!/bin/bash
set -x
env

PROMCONF=/opt/prometheus/conf
MARATHON_URL=$1

cat $PROMCONF/prometheus.yaml.tmpl | sed -e "s#MARATHON_URL#${MARATHON_URL}#g" > $PROMCONF/prometheus.yaml
/go/bin/prometheus_mesos_sd -logtostderr -mesos-exporter /go/bin/mesos-exporter -file.master $PROMCONF/mesos-groups/master.json -file.slaves $PROMCONF/mesos-groups/slaves.json $2 &
/go/bin/prometheus -config.file $PROMCONF/prometheus.yaml -web.console.templates /opt/prometheus/conf/consoles -web.console.libraries /opt/prometheus/conf/cnosoles/lib $PROMFLAGS
