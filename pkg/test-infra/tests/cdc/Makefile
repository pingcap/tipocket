# Enable GO111MODULE=on explicitly, disable it with GO111MODULE=off when necessary.
export GO111MODULE := on
export GOPRIVATE="github.com/pingcap/chaos-mesh"

test: test-build
	./run-test.sh

test-build:
	go build -o bin/ginkgo github.com/onsi/ginkgo/ginkgo
	go test -c -o bin/cdc.test .
