FROM        quay.io/prometheus/busybox:latest
MAINTAINER  The Prometheus Authors <prometheus-developers@googlegroups.com>

COPY jenkins_exporter /bin/jenkins_exporter

ENTRYPOINT ["/bin/jenkins_exporter"]
EXPOSE     9118
