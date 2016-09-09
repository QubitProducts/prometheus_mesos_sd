#!/bin/bash
set -x
env

PROMCONF=/opt/prometheus/conf
MARATHON_URL=$1

if [ -z "$CLUSTER" ]; then
  CLUSTER="default"
fi

if [ ! -z "$MESOS_STORAGE" -a ! -z "$MESOS_SANDBOX" ]; then
  PROMFLAGS="$PROMFLAGS -storage.local.path	$MESOS_SANDBOX/$MESOS_STORAGE"
fi

cat $PROMCONF/prometheus.yaml.tmpl | sed -e "s#MARATHON_URL#${MARATHON_URL}#g;s#CLUSTER#${CLUSTER}#g" > $PROMCONF/prometheus.yaml
if [ -d $PROMCONF/conf.d ]; then
  for F in $(ls $PROMCONF/conf.d/*.yaml); do
    cat $F >> $PROMCONF/prometheus.yaml
  done
fi

/go/bin/prometheus_mesos_sd -logtostderr -file.master $PROMCONF/mesos-groups/master.json -file.slaves $PROMCONF/mesos-groups/slaves.json $2 &
/go/bin/prometheus -config.file $PROMCONF/prometheus.yaml -web.console.templates /opt/prometheus/conf/consoles -web.console.libraries /opt/prometheus/conf/cnosoles/lib $PROMFLAGS
