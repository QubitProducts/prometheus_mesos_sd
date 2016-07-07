# Prometheus service discovery helped for Mesos

This is a tool for enabling prometheus (http://prometheus.io) to automatically discover a mesos and marathon cluster.

The container will launch a mesos instance, and mesos-exporter instances that are scraped locally. Your mesos
master and slaves do not need to have mesos-exporter running.


