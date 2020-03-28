GOARCH := $(if $(GOARCH),$(GOARCH),amd64)
GO=GO15VENDOREXPERIMENT="1" CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) GO111MODULE=on go
GOTEST=GO15VENDOREXPERIMENT="1" CGO_ENABLED=1 GO111MODULE=on go test # go race detector requires cgo
VERSION   := $(if $(VERSION),$(VERSION),latest)

PACKAGES := go list ./...| grep -vE 'vendor'
PACKAGE_DIRECTORIES := $(PACKAGES) | sed 's|github.com/pingcap/tipocket/||'

LDFLAGS += -X "github.com/pingcap/tipocket/cmd/util.BuildTS=$(shell date -u '+%Y-%m-%d %I:%M:%S')"
LDFLAGS += -X "github.com/pingcap/tipocket/cmd/util.BuildHash=$(shell git rev-parse HEAD)"

GOBUILD=$(GO) build -ldflags '$(LDFLAGS)'

DOCKER_REGISTRY_PREFIX := $(if $(DOCKER_REGISTRY),$(DOCKER_REGISTRY)/,)

default: build

all: build

chaos: tidb

build: fmt tidb pocket tpcc ledger txn-rand-pessimistic on-dup sqllogic block-writer \
		region-available deadlock-detector crud bank bank2 abtest cdc-pocket tiflash-pocket

tidb:
	$(GOBUILD) $(GOMOD) -o bin/chaos-tidb cmd/tidb/main.go

tpcc:
	$(GOBUILD) $(GOMOD) -o bin/tpcc cmd/tpcc/main.go

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

ledger:
	$(GOBUILD) $(GOMOD) -o bin/ledger cmd/ledger/*.go

bank:
	$(GOBUILD) $(GOMOD) -o bin/bank cmd/bank/*.go

bank2:
	$(GOBUILD) $(GOMOD) -o bin/bank2 cmd/bank2/*.go

txn-rand-pessimistic:
	$(GOBUILD) $(GOMOD) -o bin/txn-rand-pessimistic cmd/txn-rand-pessimistic/*.go

on-dup:
	$(GOBUILD) $(GOMOD) -o bin/on-dup cmd/on-dup/*.go

block-writer:
	$(GOBUILD) $(GOMOD) -o bin/block-writer cmd/block-writer/*.go

sqllogic:
	$(GOBUILD) $(GOMOD) -o bin/sqllogic cmd/sqllogic/*.go

region-available:
	$(GOBUILD) $(GOMOD) -o bin/region-available cmd/region-available/*.go

deadlock-detector:
	$(GOBUILD) $(GOMOD) -o bin/deadlock-detector cmd/deadlock-detector/*.go

crud:
	$(GOBUILD) $(GOMOD) -o bin/crud cmd/crud/*.go

abtest:
	$(GOBUILD) $(GOMOD) -o bin/abtest cmd/abtest/*.go

cdc-pocket:
	$(GOBUILD) $(GOMOD) -o bin/cdc-pocket cmd/cdc-pocket/*.go

tiflash-pocket:
	$(GOBUILD) $(GOMOD) -o bin/tiflash-pocket cmd/tiflash-pocket/*.go

fmt: groupimports
	go fmt ./...

tidy:
	@echo "go mod tidy"
	GO111MODULE=on go mod tidy
	@git diff --exit-code -- go.mod

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

image:
	docker build -t ${DOCKER_REGISTRY_PREFIX}pingcap/tipocket:latest .

docker-push:
	docker push "${DOCKER_REGISTRY_PREFIX}pingcap/tipocket:latest

.PHONY: all clean pocket compare test fmt
