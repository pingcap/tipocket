module github.com/pingcap/tipocket/pocket

go 1.13

require (
	github.com/go-sql-driver/mysql v1.4.1
	github.com/juju/errors v0.0.0-20190930114154-d42613fe1ab9
	github.com/ngaut/log v0.0.0-20180314031856-b8e36e7ba5ac
	github.com/pingcap/parser v0.0.0-20191204131342-259c92691fa4 // indirect
	github.com/pingcap/tidb v1.1.0-beta.0.20191106105829-1b72ce5987b3
	github.com/pingcap/tipb v0.0.0-20191203131953-a35f738b4796 // indirect
	github.com/pingcap/tipocket/go-sqlsmith v0.0.0-00010101000000-000000000000
	github.com/shirou/gopsutil v2.19.11+incompatible // indirect
	github.com/stretchr/testify v1.4.0
	golang.org/x/sys v0.0.0-20191204072324-ce4227a45e2e // indirect
	golang.org/x/tools v0.0.0-20191205060818-73c7173a9f7d // indirect
)

replace github.com/pingcap/tidb => github.com/you06/tidb v1.1.0-beta.0.20191107083526-0edcbcc52610

replace github.com/pingcap/tipocket/go-sqlsmith => github.com/you06/tipocket/go-sqlsmith v0.0.0-20191209063433-0cdfd1ff9a02
