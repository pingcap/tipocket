GO := CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go

.PHONY: run fmt

testbed:
	TESTBED_INPUT_SPEC=$$(yq eval -j -I=0 testbed.yaml) \
	TESTBED_OUTPUT_SPEC='{"output":"output", "items":[]}' testbed create -s http://172.16.4.180:30007
	sed -i "s|REPLACE_ME_WITH_TESTBED_TC|$$(cat output | jq -c '.items[0]')|g" .env
	sed -i "s|REPLACE_ME_WITH_TESTBED_BR|$$(cat output | jq -c '.items[1]')|g" .env
	sed -i "s|REPLACE_ME_WITH_TESTBED_GO_TPC|$$(cat output | jq -c '.items[2]')|g" .env

destroy-testbed: install-testbed-cli
	TESTBED_SETUP_OUTPUT=$$(cat output) testbed delete -s http://172.16.4.180:30007

run:
	KUBECONFIG=$(KUBECONFIG) MYSQL_PROXY=socks5://172.16.4.180:30081 go run main/cmd.go

build:
	$(GO) build -o bin/example main.go

image:
	GOOS=linux GOARCH=amd64 make build
	docker build -t hub.pingcap.net/qa/sdk_ticase_4885 .

fmt: groupimports
	go fmt ./...

groupimports: install-goimports
	goimports -w -l -local github.com/pingcap/test-infra .

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

install-testbed-cli:
	@{ \
	set -e ;\
	cd ../../services/rms/cmd ;\
	go install ./... ;\
	}