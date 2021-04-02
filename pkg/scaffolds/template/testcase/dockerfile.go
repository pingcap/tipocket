// Copyright 2021 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package testcase

import "github.com/pingcap/tipocket/pkg/scaffolds/file"

// GoModule uses for go.mod
type Dockerfile struct {
	file.TemplateMixin
	CaseName string
}

// GetIfExistsAction ...
func (g *Dockerfile) GetIfExistsAction() file.IfExistsAction {
	return file.IfExistsActionError
}

// Validate ...
func (g *Dockerfile) Validate() error {
	return g.TemplateMixin.Validate()
}

// SetTemplateDefaults ...
func (g *Dockerfile) SetTemplateDefaults() error {
	g.TemplateBody = dockerfileTemplate
	return nil
}

const dockerfileTemplate = `# syntax = docker/dockerfile:1.0-experimental
FROM golang:alpine3.10 AS build_base

RUN apk add --no-cache gcc libc-dev make bash git

ENV GO111MODULE=on
WORKDIR /src
COPY . .
COPY .git .git

RUN rm -rf /go/src/
RUN --mount=type=cache,id=tipocket_go_pkg,target=/go/pkg \
    --mount=type=cache,id=tipocket_go_cache,target=/root/.cache/go-build \
    --mount=type=tmpfs,id=tipocket_go_src,target=/go/src/ cd testcase/{{ .CaseName }} && make build

FROM alpine:3.8

RUN apk update && apk upgrade && \
    apk add --no-cache bash curl wget

RUN mkdir -p /config && mkdir -p /resources
COPY --from=0 /src/testcase/{{ .CaseName }}/bin/* /bin/
COPY --from=0 /src/config /config
COPY --from=0 /src/resources /resources

EXPOSE 8080
`
