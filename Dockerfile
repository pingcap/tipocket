# syntax = docker/dockerfile:1.0-experimental
FROM golang:alpine3.10 AS build_base

RUN apk add --no-cache gcc libc-dev make bash git

ENV GO111MODULE=on
WORKDIR /src

COPY . .
COPY .git .git

RUN --mount=type=cache,target=/go/pkg \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=tmpfs,target=/go/src/ make clean && make build

FROM alpine:3.8

RUN apk update && apk upgrade && \
    apk add --no-cache bash curl wget

RUN mkdir -p /config && mkdir -p /resources
COPY --from=0 /src/bin/* /bin/
COPY --from=0 /src/config /config
COPY --from=0 /src/resources /resources

EXPOSE 8080