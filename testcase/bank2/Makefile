
GOARCH := $(if $(GOARCH),$(GOARCH),amd64)
GO=GO15VENDOREXPERIMENT="1" CGO_ENABLED=1 GOOS=$(GOOS) GOARCH=$(GOARCH) GO111MODULE=on go
GOTEST=GO15VENDOREXPERIMENT="1" CGO_ENABLED=1 GO111MODULE=on go test # go race detector requires cgo
VERSION   := $(if $(VERSION),$(VERSION),latest)

PACKAGES := go list ./...| grep -vE 'vendor'

LDFLAGS += -s -w
LDFLAGS += -X "github.com/pingcap/tipocket/pkg/test-infra/fixture.BuildTS=$(shell date -u '+%Y-%m-%d %I:%M:%S')"
LDFLAGS += -X "github.com/pingcap/tipocket/pkg/test-infra/fixture.BuildHash=$(shell (git rev-parse --git-dir > /dev/null 2>&1 && git rev-parse HEAD) || echo 'NO-HASH')"

GOBUILD=$(GO) build -ldflags '$(LDFLAGS)'

DOCKER_REGISTRY_PREFIX := $(if $(DOCKER_REGISTRY),$(DOCKER_REGISTRY)/,)

default: tidy fmt lint build

build: mod-sum bank2

bank2:
	$(GOBUILD) $(GOMOD) -o bin/bank2 cmd/*.go

fmt: groupimports
	go fmt ./...

mod-sum:
	$(GO) mod tidy

tidy:
	@echo "go mod tidy"
	GO111MODULE=on go mod tidy
	@git rev-parse --git-dir > /dev/null 2>&1 && git diff --exit-code -- go.mod

lint: revive
	@echo "linting"
	revive -formatter friendly -config revive.toml $$($(PACKAGES))

revive:
ifeq (,$(shell which revive))
	@echo "installing revive"
	$(GO) get github.com/mgechev/revive@v1.0.2
endif

groupimports: install-goimports
	goimports -w -l -local github.com/pingcap/tipocket .

install-goimports:
ifeq (,$(shell which goimports))
	@echo "installing goimports"
	go get golang.org/x/tools/cmd/goimports
endif

clean:
	@rm -rf bin/*

test:
	$(GOTEST) ./...

.PHONY: all clean build
