GOARCH := $(if $(GOARCH),$(GOARCH),amd64)
GO=GO15VENDOREXPERIMENT="1" CGO_ENABLED=1 GOOS=$(GOOS) GOARCH=$(GOARCH) GO111MODULE=on go
GOTEST=GO15VENDOREXPERIMENT="1" CGO_ENABLED=1 GO111MODULE=on go test # go race detector requires cgo
VERSION   := $(if $(VERSION),$(VERSION),latest)

PACKAGES := go list ./...| grep -vE 'vendor'
PACKAGE_DIRECTORIES := $(PACKAGES) | sed 's|github.com/pingcap/tipocket/||'

LDFLAGS += -s -w
LDFLAGS += -X "github.com/pingcap/tipocket/pkg/test-infra/fixture.BuildTS=$(shell date -u '+%Y-%m-%d %I:%M:%S')"
LDFLAGS += -X "github.com/pingcap/tipocket/pkg/test-infra/fixture.BuildHash=$(shell git rev-parse HEAD)"

GOBUILD=$(GO) build -ldflags '$(LDFLAGS)'

DOCKER_REGISTRY_PREFIX := $(if $(DOCKER_REGISTRY),$(DOCKER_REGISTRY)/,)

default: tidy fmt lint build

build: consistency isolation pocket on-dup sqllogic block-writer \
		region-available crud \
		read-stress follower-read pessimistic resolve-lock cdc-bank \
    example \
# +tipocket:scaffold:makefile_build

consistency: bank bank2 pbank vbank ledger rawkv-linearizability tpcc pessimistic cdc-bank

isolation: list-append register

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
	cd testcase/rawkv-linearizability; make build; \
	cp bin/* ../../bin/

tpcc:
	$(GOBUILD) $(GOMOD) -o bin/tpcc cmd/tpcc/main.go

register:
	$(GOBUILD) $(GOMOD) -o bin/register cmd/register/main.go

rawkv:
	$(GOBUILD) $(GOMOD) -o bin/chaos-rawkv cmd/rawkv/main.go

txnkv:
	$(GOBUILD) $(GOMOD) -o bin/chaos-txnkv cmd/txnkv/main.go

verifier:
	$(GOBUILD) $(GOMOD) -o bin/chaos-verifier cmd/verifier/main.go

pocket:
	cd testcase/pocket; make build; \
	cp bin/* ../../bin/

compare:
	$(GOBUILD) $(GOMOD) -o bin/compare cmd/compare/*.go

on-dup:
	cd testcase/ondup; make build; \
	cp bin/* ../../bin/

block-writer:
	$(GOBUILD) $(GOMOD) -o bin/block-writer cmd/block-writer/*.go

sqllogic:
	cd testcase/sqllogictest; make build; \
	cp bin/* ../../bin/

region-available:
	$(GOBUILD) $(GOMOD) -o bin/region-available cmd/region-available/*.go

pessimistic:
	cd testcase/pessimistic; make build; \
	cp bin/* ../../bin/

crud:
	$(GOBUILD) $(GOMOD) -o bin/crud cmd/crud/*.go

cdc-bank:
	$(GOBUILD) $(GOMOD) -o bin/cdc-bank cmd/cdc-bank/*.go

read-stress:
	$(GOBUILD) $(GOMOD) -o bin/read-stress cmd/read-stress/*.go

follower-read:
	$(GOBUILD) $(GOMOD) -o bin/follower-read cmd/follower-read/*.go

titan:
	cd testcase/titan; make build; \
	cp bin/* ../../bin/

resolve-lock:
	cd testcase/resolve-lock ; make build; \
	cp bin/* ../../bin/

pipelined-locking:
	$(GOBUILD) $(GOMOD) -o bin/pipelined-locking cmd/pipelined-pessimistic-locking/*.go

example:
	cd testcase/example ; make build; \
	cp bin/* ../../bin/

list-append:
	cd testcase/list-append ; make build; \
	cp bin/* ../../bin/

# +tipocket:scaffold:makefile_build_cmd

tipocket:
	$(GOBUILD) $(GOMOD) -o bin/tipocket cmd/tipocket/*.go

fmt: groupimports
	go fmt ./...
	find testcase -mindepth 1 -maxdepth 1 -type d | xargs -I% sh -c 'cd %; make fmt';

tidy:
	@echo "go mod tidy"
	GO111MODULE=on go mod tidy
	@git diff --exit-code -- go.mod
	find testcase -mindepth 1 -maxdepth 1 -type d | xargs -I% sh -c 'cd %; make tidy';

lint: install-revive
	@echo "linting"
	revive -formatter friendly -config revive.toml $$($(PACKAGES) | grep -v "pkg/tidb-operator")
	find testcase -mindepth 1 -maxdepth 1 -type d | xargs -I% sh -c 'cd %; make lint';

install-revive:
ifeq (, $(shell which revive))
	@{ \
	set -e ;\
	TMP_DIR=$$(mktemp -d) ;\
	cd $$TMP_DIR ;\
	GO111MODULE=on go get github.com/mgechev/revive@v1.0.2 ;\
	rm -rf $$TMP_DIR ;\
	}
endif

groupimports: install-goimports
	goimports -w -l -local github.com/pingcap/tipocket $$($(PACKAGE_DIRECTORIES))

init: tipocket
	bin/tipocket init -c $(c)

install-goimports:
ifeq (, $(shell which goimports))
	@{ \
	set -e ;\
	TMP_DIR=$$(mktemp -d) ;\
	cd $$TMP_DIR ;\
	GO111MODULE=on go get golang.org/x/tools/cmd/goimports ;\
	rm -rf $$TMP_DIR ;\
	}
endif

install-jsonnet:
ifeq (, $(shell which jsonnet))
	@{ \
	set -e ;\
	TMP_DIR=$$(mktemp -d) ;\
	cd $$TMP_DIR ;\
	GO111MODULE=on go get github.com/google/go-jsonnet/cmd/jsonnet ;\
	rm -rf $$TMP_DIR ;\
	}
endif

install-jsonnetfmt:
ifeq (, $(shell which jsonnetfmt))
	@{ \
	set -e ;\
	TMP_DIR=$$(mktemp -d) ;\
	cd $$TMP_DIR ;\
	GO111MODULE=on go get github.com/google/go-jsonnet/cmd/jsonnetfmt ;\
	rm -rf $$TMP_DIR ;\
	}
endif

install-yq:
ifeq (, $(shell which yq))
	@{ \
	set -e ;\
	TMP_DIR=$$(mktemp -d) ;\
	cd $$TMP_DIR ;\
	GO111MODULE=on go get github.com/mikefarah/yq/v4@v4.4.1 ;\
	rm -rf $$TMP_DIR ;\
	}
endif

install-jb:
ifeq (, $(shell which jb))
	@{ \
	set -e ;\
	TMP_DIR=$$(mktemp -d) ;\
	cd $$TMP_DIR ;\
	GO111MODULE=on go get github.com/jsonnet-bundler/jsonnet-bundler/cmd/jb ;\
	rm -rf $$TMP_DIR ;\
	}
endif

clean_workflow:
	rm -rf run/manifest
	mkdir -p run/manifest/workflow
	mkdir -p run/manifest/cron_workflow

fmt_workflow: install-jsonnetfmt
	find run -name "*.jsonnet" | xargs -I{} jsonnetfmt -i {}

build_workflow: fmt_workflow install-jsonnet install-yq
	mkdir -p run/manifest/workflow
	find run/workflow -name "*.jsonnet" -type f -exec basename {} \;  | xargs -I% sh -c 'jsonnet run/build.jsonnet -J run/lib -J run/workflow --ext-str build_mode="workflow" --ext-code-file file=run/workflow/% | yq eval -P - > run/manifest/workflow/%.yaml'

build_cron_workflow: fmt_workflow install-jsonnet install-yq
	mkdir -p run/manifest/cron_workflow
	find run/workflow -name "*.jsonnet" -type f -exec basename {} \;  | xargs -I% sh -c 'jsonnet run/build.jsonnet -J run/lib -J run/workflow --ext-str build_mode="cron_workflow" --ext-code-file file=run/workflow/% | yq eval -P - > run/manifest/cron_workflow/%.yaml'

clean:
	@rm -rf bin/*

test:
	$(GOTEST) ./...
	find testcase -mindepth 1 -maxdepth 1 -type d | xargs -I% sh -c 'cd %; make test';

image:
	DOCKER_BUILDKIT=1 docker build -t ${DOCKER_REGISTRY_PREFIX}pingcap/tipocket:latest .

docker-push:
	docker push ${DOCKER_REGISTRY_PREFIX}pingcap/tipocket:latest

.PHONY: all clean pocket compare test fmt
