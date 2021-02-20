GOVER_MAJOR := $(shell go version | sed -E -e "s/.*go([0-9]+)[.]([0-9]+).*/\1/")
GOVER_MINOR := $(shell go version | sed -E -e "s/.*go([0-9]+)[.]([0-9]+).*/\2/")
GO113 := $(shell [ $(GOVER_MAJOR) -gt 1 ] || [ $(GOVER_MAJOR) -eq 1 ] && [ $(GOVER_MINOR) -ge 13 ]; echo $$?)
ifeq ($(GO113), 1)
$(error Please upgrade your Go compiler to 1.13 or higher version)
endif

# Enable GO111MODULE=on explicitly, disable it with GO111MODULE=off when necessary.
export GO111MODULE := on
GOARCH := $(if $(GOARCH),$(GOARCH),amd64)
GOENV  := GO15VENDOREXPERIMENT="1" CGO_ENABLED=0 GOARCH=$(GOARCH)
GO     := $(GOENV) go
GO_BUILD := $(GO) build -trimpath

PACKAGES := go list ./...| grep -vE 'vendor'

LDFLAGS += -s -w
LDFLAGS += -X "github.com/pingcap/tipocket/pkg/test-infra/fixture.BuildTS=$(shell date -u '+%Y-%m-%d %I:%M:%S')"
LDFLAGS += -X "github.com/pingcap/tipocket/pkg/test-infra/fixture.BuildHash=$(shell git rev-parse HEAD)"

GOBUILD=$(GO) build -ldflags '$(LDFLAGS)'

DOCKER_REGISTRY_PREFIX := $(if $(DOCKER_REGISTRY),$(DOCKER_REGISTRY)/,)

default: tidy fmt lint build

build: cross_region

linux:
	GOOS=linux $(GOBUILD) $(GOMOD) -o bin/cross-region-linux cmd/*.go

cross_region:
	$(GOBUILD) $(GOMOD) -o bin/cross-region cmd/*.go

fmt: groupimports
	go fmt ./...

tidy:
	@echo "go mod tidy"
	GO111MODULE=on go mod tidy
	@git diff --exit-code -- go.mod

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
