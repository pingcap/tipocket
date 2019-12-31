GOOS := $(if $(GOOS),$(GOOS),linux)
GOARCH := $(if $(GOARCH),$(GOARCH),amd64)
GO=GO15VENDOREXPERIMENT="1" CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) GO111MODULE=on go
GOTEST=GO15VENDOREXPERIMENT="1" CGO_ENABLED=1 GO111MODULE=on go test # go race detector requires cgo
VERSION   := $(if $(VERSION),$(VERSION),latest)

PACKAGES := $$(go list ./...| grep -vE 'vendor' )

FILES     := $$(find . -name "*.go" | grep -vE "vendor")
GOFILTER  := grep -vE 'vendor|render.Delims|bindata_assetfs|testutil|\.pb\.go'
GOCHECKER := $(GOFILTER) | awk '{ print } END { if (NR > 0) { exit 1 } }'
GOLINT    := go list ./... | grep -vE 'vendor' | xargs -L1 -I {} golint {} 2>&1 | $(GOCHECKER)

LDFLAGS += -X "github.com/pingcap/tipocket/pocket/util.BuildTS=$(shell date -u '+%Y-%m-%d %I:%M:%S')"
LDFLAGS += -X "github.com/pingcap/tipocket/pocket/util.BuildHash=$(shell git rev-parse HEAD)"

GOBUILD=$(GO) build -ldflags '$(LDFLAGS)'

default: build

build: pocket

pocket:
	$(GOBUILD) $(GOMOD) -o bin/pocket cmd/pocket/*.go

clean:
	@rm -rf bin/*

test:
	$(GOTEST) ./...

.PHONY: all clean pocket test
