#!/bin/bash
set -x

PROMCONF=/opt/prometheus/conf
MARATHON_URL=$1

cat $PROMCONF/prometheus.yaml.tmpl | sed -e "s#MARATHON_URL#${MARATHON_URL}#g" > $PROMCONF/prometheus.yaml
/go/bin/prometheus_mesos_sd -logtostderr -mesos-exporter /go/bin/mesos-exporter -file.master $PROMCONF/mesos-groups/master.json -file.slaves $PROMCONF/mesos-groups/slaves.json $2 &
#/go/bin/alertmanager -config $PROMCONF/alertmanager.yaml &
/go/bin/prometheus -config.file $PROMCONF/prometheus.yaml
