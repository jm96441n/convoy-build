ARG ENVOY_VERSION=v1.26-latest
FROM envoyproxy/envoy:${ENVOY_VERSION}


ENV CONSUL_HTTP_ADDR=http://localhost:8500

RUN apt-get update && \
    apt-get install -y \
      bash \
      curl \
      jq && \
    rm -rf /var/lib/apt/lists/*

COPY ./consul /bin/consul

COPY ./entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
