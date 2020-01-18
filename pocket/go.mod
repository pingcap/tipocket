module github.com/pingcap/tipocket/pocket

go 1.13

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/StackExchange/wmi v0.0.0-20190523213315-cbe66965904d // indirect
	github.com/go-ole/go-ole v1.2.4 // indirect
	github.com/go-sql-driver/mysql v1.4.1
	github.com/juju/errors v0.0.0-20190930114154-d42613fe1ab9
	github.com/juju/loggo v0.0.0-20190526231331-6e530bcce5d8 // indirect
	github.com/juju/testing v0.0.0-20191001232224-ce9dec17d28b // indirect
	github.com/mgechev/revive v1.0.1 // indirect
	github.com/ngaut/log v0.0.0-20180314031856-b8e36e7ba5ac
	github.com/opentracing/opentracing-go v1.1.0 // indirect
	github.com/pingcap/tidb v1.1.0-beta.0.20191106105829-1b72ce5987b3
	github.com/pingcap/tipocket/go-sqlsmith v0.0.0-20191209122549-dc7dbc1100a3
	github.com/remyoudompheng/bigfft v0.0.0-20190728182440-6a916e37a237 // indirect
	github.com/shirou/gopsutil v2.19.11+incompatible // indirect
	github.com/stretchr/testify v1.4.0
	github.com/you06/sqlsmith-go v0.0.0-20191205065339-ef0a8db01d04 // indirect
	go.uber.org/atomic v1.5.1 // indirect
	go.uber.org/multierr v1.4.0 // indirect
	go.uber.org/zap v1.13.0 // indirect
	golang.org/x/lint v0.0.0-20191125180803-fdd1cda4f05f // indirect
	golang.org/x/net v0.0.0-20191207000613-e7e4b65ae663 // indirect
	golang.org/x/tools v0.0.0-20200114052453-d31a08c2edf2 // indirect
	gopkg.in/mgo.v2 v2.0.0-20190816093944-a6b53ec6cb22 // indirect
)

replace github.com/pingcap/tidb => github.com/you06/tidb v1.1.0-beta.0.20191107083526-0edcbcc52610

replace github.com/pingcap/tipocket/go-sqlsmith => ../go-sqlsmith
