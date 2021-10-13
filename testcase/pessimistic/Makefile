GOARCH := $(if $(GOARCH),$(GOARCH),amd64)
GO=GO15VENDOREXPERIMENT="1" CGO_ENABLED=1 GOOS=$(GOOS) GOARCH=$(GOARCH) GO111MODULE=on go
GOTEST=GO15VENDOREXPERIMENT="1" CGO_ENABLED=1 GO111MODULE=on go test # go race detector requires cgo
VERSION   := $(if $(VERSION),$(VERSION),latest)

PACKAGES := go list ./...| grep -vE 'vendor'

LDFLAGS += -X "github.com/pingcap/tipocket/pkg/test-infra/fixture.BuildTS=$(shell date -u '+%Y-%m-%d %I:%M:%S')"
LDFLAGS += -X "github.com/pingcap/tipocket/pkg/test-infra/fixture.BuildHash=$(shell (git rev-parse --git-dir > /dev/null 2>&1 && git rev-parse HEAD) || echo 'NO-HASH')"

GOBUILD=$(GO) build -ldflags '$(LDFLAGS)'

DOCKER_REGISTRY_PREFIX := $(if $(DOCKER_REGISTRY),$(DOCKER_REGISTRY)/,)

default: tidy fmt lint build

build: deadlock-detector txn-rand-pessimistic pipelined-pessimistic-locking

deadlock-detector:
	$(GOBUILD) $(GOMOD) -o bin/deadlock-detector cmd/deadlock-detector/*.go

txn-rand-pessimistic:
	$(GOBUILD) $(GOMOD) -o bin/txn-rand-pessimistic cmd/txn-rand-pessimistic/*.go

pipelined-pessimistic-locking:
	$(GOBUILD) $(GOMOD) -o bin/pipelined-pessimistic-locking cmd/pipelined-pessimistic-locking/*.go

fmt: groupimports
	go fmt ./...

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
