FROM golang

ENV PROM_VERSION 1.0.1
ENV PROM_SHA256 2ef4f7e416c6fdc07396be9a72f69670733a0c6f72060c8bb42d6ff3410eae96

ENV AM_VERSION 0.3.0
ENV AM_SHA256 ee5ce1aa3e6d4602e112ae9fe7dc3ddf8f80bea0d96dc74861b90d947a3d9c32

RUN mkdir -p /opt/prometheus/conf/mesos-groups
RUN mkdir -p /opt/prometheus/conf/rules
RUN mkdir -p /opt/prometheus/conf/alerts

RUN wget https://github.com/prometheus/prometheus/releases/download/v${PROM_VERSION}/prometheus-${PROM_VERSION}.linux-amd64.tar.gz
RUN wget https://github.com/prometheus/alertmanager/releases/download/v${AM_VERSION}/alertmanager-${AM_VERSION}.linux-amd64.tar.gz
RUN echo "${PROM_SHA256}  prometheus-${PROM_VERSION}.linux-amd64.tar.gz" >> sums
RUN echo "${AM_SHA256}  alertmanager-${AM_VERSION}.linux-amd64.tar.gz" >> sums
RUN sha256sum -c -w --strict sums
RUN tar xvzf prometheus-${PROM_VERSION}.linux-amd64.tar.gz
RUN cp prometheus-${PROM_VERSION}.linux-amd64/prometheus /go/bin
RUN cp prometheus-${PROM_VERSION}.linux-amd64/promtool /go/bin
RUN cp -r prometheus-${PROM_VERSION}.linux-amd64/consoles /opt/prometheus/conf/consoles
RUN cp -r prometheus-${PROM_VERSION}.linux-amd64/console_libraries /opt/prometheus/conf/consoles/libs
RUN tar xvzf alertmanager-${AM_VERSION}.linux-amd64.tar.gz
RUN cp alertmanager-${AM_VERSION}.linux-amd64/alertmanager /go/bin
RUN rm prometheus-${PROM_VERSION}.linux-amd64.tar.gz
RUN rm alertmanager-${AM_VERSION}.linux-amd64.tar.gz

RUN go get github.com/mesosphere/mesos-exporter

ADD . /go/src/github.com/QubitProducts/prometheus_mesos_sd
RUN go install github.com/QubitProducts/prometheus_mesos_sd

ADD docker_run.sh /usr/local/bin/docker_run.sh

ADD prometheus.yaml.tmpl /opt/prometheus/conf/prometheus.yaml.tmpl

ENTRYPOINT ["/usr/local/bin/docker_run.sh"]

EXPOSE 9090
