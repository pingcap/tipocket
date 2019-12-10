# Enable GO111MODULE=on explicitly, disable it with GO111MODULE=off when necessary.
export GO111MODULE := on
GOOS := $(if $(GOOS),$(GOOS),linux)
GOARCH := $(if $(GOARCH),$(GOARCH),amd64)
GOENV  := GO15VENDOREXPERIMENT="1" CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH)
GO     := $(GOENV) go
GO_BUILD := $(GO) build -trimpath

test-build:
	$(GO_BUILD) -ldflags '$(LDFLAGS)' -o images/test/bin/ginkgo github.com/onsi/ginkgo/ginkgo
	$(GO) test -c -ldflags '$(LDFLAGS)' -o images/test/bin/e2e.test ./tests
