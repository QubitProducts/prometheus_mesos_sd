FROM golang

ENV PROM_VERSION 1.6.1
ENV PROM_SHA256 07f466ad23848c7dd740e722355d25ae1d24584d6574a8f10542d1da13368d2f

ENV AM_VERSION 0.5.1
ENV AM_SHA256 9df9f0eb0061c8ead1b89060b851ea389fbdf6c1adc8513b40f6f4b90f4de932

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
RUN cp -r prometheus-${PROM_VERSION}.linux-amd64/console_libraries /opt/prometheus/conf/console_libraries
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
