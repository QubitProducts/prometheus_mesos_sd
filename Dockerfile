FROM golang

ENV PROM_VERSION 1.1.2
ENV PROM_SHA256 dc97178c46ca04a1a1306037979d9157f758163915c78bbe07eb3d234e254867

ENV AM_VERSION 0.4.2
ENV AM_SHA256 8d04e3dc9955a0a3897d126c996b921cfd1dc80ff6ff982936ce8f703baa2eb8

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
