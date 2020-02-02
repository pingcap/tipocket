module github.com/pingcap/tipocket/go-sqlsmith

go 1.13

require (
	github.com/juju/errors v0.0.0-20190930114154-d42613fe1ab9
	github.com/ngaut/log v0.0.0-20180314031856-b8e36e7ba5ac
	github.com/pingcap/parser v0.0.0-20200109073933-a9496438d77d
	github.com/pingcap/tidb v1.1.0-beta.0.20200110034112-1b34cc234e82
	github.com/satori/go.uuid v1.2.0
	github.com/stretchr/testify v1.4.0
)

replace github.com/pingcap/tipocket/pocket => ../pocket
