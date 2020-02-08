module github.com/pingcap/tipocket

go 1.13

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/anishathalye/porcupine v0.0.0-20190205033716-f6fec466e840
	github.com/go-sql-driver/mysql v1.4.1
	github.com/juju/errors v0.0.0-20190930114154-d42613fe1ab9
	github.com/ngaut/log v0.0.0-20180314031856-b8e36e7ba5ac
	github.com/pingcap/chaos-mesh v0.0.0-20200208111039-5f0dd626bd40

	github.com/pingcap/parser v0.0.0-20200109073933-a9496438d77d
	github.com/pingcap/tidb v2.1.0-beta+incompatible
	github.com/pingcap/tipocket/test-infra v0.0.0-20200207052846-4ecf77c065cd

	github.com/satori/go.uuid v1.2.0
	github.com/stretchr/testify v1.4.0
)

replace github.com/pingcap/tidb => github.com/you06/tidb v1.1.0-beta.0.20191107083526-0edcbcc52610

replace github.com/pingcap/tipocket/pkg/go-sqlsmith => ../go-sqlsmith

replace k8s.io/api => k8s.io/api v0.0.0-20190918155943-95b840bb6a1f

replace k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20190918161926-8f644eb6e783

replace k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190913080033-27d36303b655

replace k8s.io/apiserver => k8s.io/apiserver v0.0.0-20190918160949-bfa5e2e684ad

replace k8s.io/cli-runtime => k8s.io/cli-runtime v0.0.0-20190918162238-f783a3654da8

replace k8s.io/client-go => k8s.io/client-go v0.0.0-20190918160344-1fbdaa4c8d90

replace k8s.io/code-generator => k8s.io/code-generator v0.0.0-20190912054826-cd179ad6a269

replace k8s.io/csi-api => k8s.io/csi-api v0.0.0-20190118125032-c4557c74373f

replace k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.0.0-20190918161219-8c8f079fddc3

replace k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.0.0-20190918162944-7a93a0ddadd8

replace k8s.io/kube-proxy => k8s.io/kube-proxy v0.0.0-20190918162534-de037b596c1e

replace k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.0.0-20190918162820-3b5c1246eb18

replace k8s.io/kubelet => k8s.io/kubelet v0.0.0-20190918162654-250a1838aa2c

replace k8s.io/metrics => k8s.io/metrics v0.0.0-20190918162108-227c654b2546

replace k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.0.0-20190918161442-d4c9c65c82af

replace k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.0.0-20190918162410-e45c26d066f2

replace k8s.io/sample-controller => k8s.io/sample-controller v0.0.0-20190918161628-92eb3cb7496c

replace k8s.io/cloud-provider => k8s.io/cloud-provider v0.0.0-20190918163234-a9c1f33e9fb9

replace k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.0.0-20190918163108-da9fdfce26bb

replace k8s.io/component-base => k8s.io/component-base v0.0.0-20190918160511-547f6c5d7090

replace k8s.io/cri-api => k8s.io/cri-api v0.0.0-20190828162817-608eb1dad4ac

replace k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.0.0-20190918163402-db86a8c7bb21

replace k8s.io/kubectl => k8s.io/kubectl v0.0.0-20190918164019-21692a0861df

replace k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.0.0-20190918163543-cfa506e53441

replace k8s.io/node-api => k8s.io/node-api v0.0.0-20190918163711-2299658ad911

replace github.com/uber-go/atomic => go.uber.org/atomic v1.5.0

replace github.com/pingcap/tipocket/pkg/pocket => ../pocket
