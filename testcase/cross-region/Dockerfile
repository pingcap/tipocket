# syntax = docker/dockerfile:1.0-experimental
FROM golang:alpine3.10 AS build_base

RUN apk update && apk add --no-cache wget bash sed make git

WORKDIR /src
COPY . .
COPY .git .git

WORKDIR /src/testcase/cross-region

RUN rm -rf /go/src/
RUN --mount=type=cache,id=tipocket_go_pkg,target=/go/pkg \
    --mount=type=cache,id=tipocket_go_cache,target=/root/.cache/go-build \
    --mount=type=tmpfs,id=tipocket_go_src,target=/go/src/ make build

FROM alpine:3.8

RUN mkdir -p /config && mkdir -p /resources
COPY --from=0 /src/config /config
COPY --from=0 /src/resources /resources
COPY --from=0 /src/testcase/cross-region/bin/cross-region /bin/cross-region
ENTRYPOINT [ "/bin/cross-region" ]
