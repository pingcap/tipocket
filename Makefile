PACKAGE_LIST := go list ./... | grep -vE "vendor"
PACKAGE_DIRECTORIES := $(PACKAGE_LIST) | sed 's|github.com/pingcap/tipocket/||'

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

fmt: groupimports
	go fmt ./...

groupimports: install-goimports
	goimports -w -l -local github.com/pingcap/tipocket $$($(PACKAGE_DIRECTORIES))

install-goimports:
ifeq (,$(shell which goimports))
	@echo "installing goimports"
	go get golang.org/x/tools/cmd/goimports
endif