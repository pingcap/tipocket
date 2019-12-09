module github.com/pingcap/tipocket/go-sqlsmith

go 1.13

require (
	github.com/juju/errors v0.0.0-20190930114154-d42613fe1ab9
	github.com/ngaut/log v0.0.0-20180314031856-b8e36e7ba5ac
	github.com/pingcap/parser v0.0.0-20191031081038-bfb0c3adf567
	github.com/pingcap/tidb v1.1.0-beta.0.20191106105829-1b72ce5987b3
	github.com/stretchr/testify v1.3.0
	github.com/you06/sqlsmith-go v0.0.0-20191205065339-ef0a8db01d04 // indirect
)

replace github.com/pingcap/tidb => github.com/you06/tidb v1.1.0-beta.0.20191107073350-1f019a46fa2c
