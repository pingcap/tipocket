FROM golang:alpine3.10 AS build_base

RUN apk add --no-cache gcc make bash git

ENV GO111MODULE=on
RUN mkdir /src
WORKDIR /src
COPY go.mod .
COPY go.sum .

RUN go mod download

FROM build_base AS binary_builder

COPY . /src
WORKDIR /src
RUN make build