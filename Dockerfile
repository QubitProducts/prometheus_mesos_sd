FROM golang

RUN go get github.com/prometheus/prometheus/cmd/prometheus
RUN go get github.com/prometheus/alertmanager
RUN go get github.com/mesosphere/mesos-exporter

ADD . /go/src/github.com/QubitProducts/prometheus_mesos_sd
RUN go install github.com/QubitProducts/prometheus_mesos_sd

ADD docker_run.sh /usr/local/bin/docker_run.sh

RUN mkdir -p /opt/prometheus/conf/mesos-groups
RUN mkdir -p /opt/prometheus/conf/rules
RUN mkdir -p /opt/prometheus/conf/alerts
ADD prometheus.yaml.tmpl /opt/prometheus/conf/prometheus.yaml.tmpl
#ADD alertmanager.yaml /opt/prometheus/conf/alertmanager.yaml


ENTRYPOINT ["/usr/local/bin/docker_run.sh"]

EXPOSE 9090
#EXPOSE 9093

