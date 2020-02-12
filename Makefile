GOARCH := $(if $(GOARCH),$(GOARCH),amd64)
GO=GO15VENDOREXPERIMENT="1" CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) GO111MODULE=on go
GOTEST=GO15VENDOREXPERIMENT="1" CGO_ENABLED=1 GO111MODULE=on go test # go race detector requires cgo
VERSION   := $(if $(VERSION),$(VERSION),latest)

PACKAGES := go list ./...| grep -vE 'vendor'
PACKAGE_DIRECTORIES := $(PACKAGES) | sed 's|github.com/pingcap/tipocket/||'

LDFLAGS += -X "github.com/pingcap/tipocket/pkg/pocket/util.BuildTS=$(shell date -u '+%Y-%m-%d %I:%M:%S')"
LDFLAGS += -X "github.com/pingcap/tipocket/pkg/pocket/util.BuildHash=$(shell git rev-parse HEAD)"

GOBUILD=$(GO) build -ldflags '$(LDFLAGS)'

default: build

all: build

build: fmt chaos verifier

chaos: tidb

tidb:
	$(GOBUILD) $(GOMOD) -o bin/chaos-tidb cmd/tidb/main.go

rawkv:
	$(GOBUILD) $(GOMOD) -o bin/chaos-rawkv cmd/rawkv/main.go

txnkv:
	$(GOBUILD) $(GOMOD) -o bin/chaos-txnkv cmd/txnkv/main.go

verifier:
	$(GOBUILD) $(GOMOD) -o bin/chaos-verifier cmd/verifier/main.go

pocket:
	$(GOBUILD) $(GOMOD) -o bin/pocket cmd/pocket/*.go

compare:
	$(GOBUILD) $(GOMOD) -o bin/compare cmd/compare/*.go

fmt: groupimports
	go fmt ./...

groupimports: install-goimports
	goimports -w -l -local github.com/pingcap/tipocket $$($(PACKAGE_DIRECTORIES))

install-goimports:
ifeq (,$(shell which goimports))
	@echo "installing goimports"
	go get golang.org/x/tools/cmd/goimports
endif

clean:
	@rm -rf bin/*

test:
	$(GOTEST) ./...

.PHONY: all clean pocket compare test fmt
