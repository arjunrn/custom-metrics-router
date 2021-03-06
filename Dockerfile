FROM ubuntu:focal

COPY build/linux/custom-metrics-router /

CMD /custom-metrics-router
