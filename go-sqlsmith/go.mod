module github.com/pingcap/tipocket/go-sqlsmith

go 1.13

require (
	github.com/juju/errors v0.0.0-20190930114154-d42613fe1ab9
	github.com/ngaut/log v0.0.0-20180314031856-b8e36e7ba5ac
	github.com/pingcap/parser v0.0.0-20191210060830-bdf23a7ade01
	github.com/pingcap/tidb v1.1.0-beta.0.20191106105829-1b72ce5987b3
	github.com/pingcap/tipb v0.0.0-20191209145133-44f75c9bef33 // indirect
	github.com/satori/go.uuid v1.2.0
	github.com/stretchr/testify v1.4.0
	golang.org/x/crypto v0.0.0-20191206172530-e9b2fee46413 // indirect
	golang.org/x/sys v0.0.0-20191210023423-ac6580df4449 // indirect
)

replace github.com/pingcap/tidb => github.com/you06/tidb v1.1.0-beta.0.20191107073350-1f019a46fa2c

replace github.com/pingcap/tipocket/pocket => ../pocket
