GOARCH := $(if $(GOARCH),$(GOARCH),amd64)
GO=GO15VENDOREXPERIMENT="1" CGO_ENABLED=1 GOOS=$(GOOS) GOARCH=$(GOARCH) GO111MODULE=on go
GOTEST=GO15VENDOREXPERIMENT="1" CGO_ENABLED=1 GO111MODULE=on go test # go race detector requires cgo
VERSION   := $(if $(VERSION),$(VERSION),latest)

PACKAGES := go list ./...| grep -vE 'vendor'
PACKAGE_DIRECTORIES := $(PACKAGES) | sed 's|github.com/pingcap/tipocket/||'

LDFLAGS += "-s -w"
LDFLAGS += -X "github.com/pingcap/tipocket/pkg/test-infra/fixture.BuildTS=$(shell date -u '+%Y-%m-%d %I:%M:%S')"
LDFLAGS += -X "github.com/pingcap/tipocket/pkg/test-infra/fixture.BuildHash=$(shell git rev-parse HEAD)"

GOBUILD=$(GO) build -ldflags '$(LDFLAGS)'

DOCKER_REGISTRY_PREFIX := $(if $(DOCKER_REGISTRY),$(DOCKER_REGISTRY)/,)

default: tidy fmt lint build

build: consistency isolation pocket on-dup sqllogic block-writer \
		region-available crud \
		read-stress follower-read pessimistic

consistency: bank bank2 pbank vbank ledger rawkv-linearizability tpcc pessimistic

isolation: append register

bank:
	$(GOBUILD) $(GOMOD) -o bin/bank cmd/bank/*.go

bank2:
	$(GOBUILD) $(GOMOD) -o bin/bank2 cmd/bank2/*.go

backup:
	$(GOBUILD) $(GOMOD) -o bin/backup cmd/backup/*.go

pbank:
	$(GOBUILD) $(GOMOD) -o bin/pbank cmd/pbank/main.go

vbank:
	$(GOBUILD) $(GOMOD) -o bin/vbank cmd/vbank/*.go

ledger:
	$(GOBUILD) $(GOMOD) -o bin/ledger cmd/ledger/*.go

rawkv-linearizability:
	cd testcase/rawkv-linearizability make build; \
	cp bin/* ../../bin/

tpcc:
	$(GOBUILD) $(GOMOD) -o bin/tpcc cmd/tpcc/main.go

append:
	$(GOBUILD) $(GOMOD) -o bin/append cmd/append/main.go

register:
	$(GOBUILD) $(GOMOD) -o bin/register cmd/register/main.go

rawkv:
	$(GOBUILD) $(GOMOD) -o bin/chaos-rawkv cmd/rawkv/main.go

txnkv:
	$(GOBUILD) $(GOMOD) -o bin/chaos-txnkv cmd/txnkv/main.go

verifier:
	$(GOBUILD) $(GOMOD) -o bin/chaos-verifier cmd/verifier/main.go

pocket:
	cd testcase/pocket make build; \
	cp bin/* ../../bin/

compare:
	$(GOBUILD) $(GOMOD) -o bin/compare cmd/compare/*.go

on-dup:
	$(GOBUILD) $(GOMOD) -o bin/on-dup cmd/on-dup/*.go

block-writer:
	$(GOBUILD) $(GOMOD) -o bin/block-writer cmd/block-writer/*.go

sqllogic:
	$(GOBUILD) $(GOMOD) -o bin/sqllogic cmd/sqllogic/*.go

region-available:
	$(GOBUILD) $(GOMOD) -o bin/region-available cmd/region-available/*.go

pessimistic:
	cd testcase/pessimistic; make build; \
	cp bin/* ../../bin/

crud:
	$(GOBUILD) $(GOMOD) -o bin/crud cmd/crud/*.go

read-stress:
	$(GOBUILD) $(GOMOD) -o bin/read-stress cmd/read-stress/*.go

follower-read:
	$(GOBUILD) $(GOMOD) -o bin/follower-read cmd/follower-read/*.go

titan:
	cd testcase/titan; make build; \
	cp bin/* ../../bin/

pipelined-locking:
	$(GOBUILD) $(GOMOD) -o bin/pipelined-locking cmd/pipelined-pessimistic-locking/*.go

fmt: groupimports
	go fmt ./...

tidy:
	@echo "go mod tidy"
	GO111MODULE=on go mod tidy
	@git diff --exit-code -- go.mod

lint: revive
	@echo "linting"
	revive -formatter friendly -config revive.toml $$($(PACKAGES) | grep -v "pkg/tidb-operator")

revive:
	$(GO) get github.com/mgechev/revive@v1.0.2

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
	DOCKER_BUILDKIT=1 docker build -t ${DOCKER_REGISTRY_PREFIX}pingcap/tipocket:latest .

docker-push:
	docker push ${DOCKER_REGISTRY_PREFIX}pingcap/tipocket:latest

.PHONY: all clean pocket compare test fmt
