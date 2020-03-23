FROM golang:alpine3.10 AS build_base

RUN apk add --no-cache gcc make bash git curl

ENV GO111MODULE=on
RUN mkdir /src
WORKDIR /src
COPY go.mod .
COPY go.sum .

COPY . .

FROM alpine:3.8

RUN apk update && apk upgrade && \
    apk add --no-cache bash curl wget

RUN mkdir -p /config
COPY --from=0 /src/bin/* /bin/
COPY --from=0 /src/configmap /config

EXPOSE 8080
