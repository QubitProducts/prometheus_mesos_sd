# Prometheus service discovery helped for Mesos

This is a tool for enabling prometheus (http://prometheus.io) to automatically discover a mesos and marathon cluster.

Each host need mesos-exporter exposed. 5051 + 9105 for mesos-slaves, and 5050 + 9110 for mesos masters. The container will locate
the current slaves and the active master via mesos.

NOTE!

An earlier version of this image is available that automatically manages mesos-exporter instances inside the container.
That is still availabl on the branch spawns_mesosexporter. This works well, but means targets page has localhost
links. That's not terrible, but in the end I preferred having the agent locally on each host.



