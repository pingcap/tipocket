FROM bitnami/kubectl:1.16
USER root

ENV LOKI_VERSION=v1.3.0
RUN apt-get update && apt-get install -y jq \
    && apt-get clean
RUN wget https://github.com/grafana/loki/releases/download/${LOKI_VERSION}/logcli-linux-amd64.zip \
    && unzip logcli-linux-amd64.zip && rm -rf logcli-linux-amd64.zip \
    && mv logcli-linux-amd64 /bin/logcli && chmod +x /bin/logcli
