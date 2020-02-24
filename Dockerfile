FROM golang:alpine3.10 AS build_base

RUN apk add --no-cache gcc make bash git curl

ENV GO111MODULE=on
RUN mkdir /src
WORKDIR /src
COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN make build

FROM alpine:3.8

RUN apk update && apk upgrade && \
    apk add --no-cache bash curl

COPY --from=0 /src/bin/* /bin/

EXPOSE 8080
