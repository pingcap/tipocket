GOOS := $(if $(GOOS),$(GOOS),linux)
GOARCH := $(if $(GOARCH),$(GOARCH),amd64)
GO=GO15VENDOREXPERIMENT="1" CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) GO111MODULE=on go
GOTEST=GO15VENDOREXPERIMENT="1" CGO_ENABLED=1 GO111MODULE=on go test # go race detector requires cgo
VERSION   := $(if $(VERSION),$(VERSION),latest)

PACKAGES := go list ./...| grep -vE 'vendor'
PACKAGE_DIRECTORIES := $(PACKAGES) | sed 's|github.com/pingcap/tipocket/||'

FILES     := $$(find . -name "*.go" | grep -vE "vendor")
GOFILTER  := grep -vE 'vendor|render.Delims|bindata_assetfs|testutil|\.pb\.go'
GOCHECKER := $(GOFILTER) | awk '{ print } END { if (NR > 0) { exit 1 } }'
GOLINT    := go list ./... | grep -vE 'vendor' | xargs -L1 -I {} golint {} 2>&1 | $(GOCHECKER)

LDFLAGS += -X "github.com/pingcap/tipocket/pocket/util.BuildTS=$(shell date -u '+%Y-%m-%d %I:%M:%S')"
LDFLAGS += -X "github.com/pingcap/tipocket/pocket/util.BuildHash=$(shell git rev-parse HEAD)"

GOBUILD=$(GO) build -ldflags '$(LDFLAGS)'

default: build

all: build

build: fmt chaos verifier

chaos: rawkv tidb txnkv

tidb:
	GO111MODULE=on go build -o bin/chaos-tidb cmd/tidb/main.go

rawkv:
	GO111MODULE=on go build -o bin/chaos-rawkv cmd/rawkv/main.go

txnkv:
	GO111MODULE=on go build -o bin/chaos-txnkv cmd/txnkv/main.go

verifier:
	GO111MODULE=on go build -o bin/chaos-verifier cmd/verifier/main.go

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
