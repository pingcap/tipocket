module github.com/pingcap/tipocket

go 1.13

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/anishathalye/porcupine v0.0.0-20200229220004-848b8b5d43d9
	github.com/cznic/mathutil v0.0.0-20181122101859-297441e03548
	github.com/gengliqi/persistent_treap v0.0.0-20200403155416-2b2a1532211c
	github.com/go-openapi/jsonreference v0.19.3 // indirect
	github.com/go-openapi/spec v0.19.4 // indirect
	github.com/go-sql-driver/mysql v1.4.1
	github.com/google/uuid v1.1.1
	github.com/grafana/loki v1.3.1-0.20200316172301-1eb139c37c1c
	github.com/juju/errors v0.0.0-20190930114154-d42613fe1ab9
	github.com/ngaut/log v0.0.0-20180314031856-b8e36e7ba5ac
	github.com/pingcap/advanced-statefulset v0.3.2
	github.com/pingcap/chaos-mesh v0.0.0-20200221071630-a3e79a893072
	github.com/pingcap/errors v0.11.5-0.20190809092503-95897b64e011
	github.com/pingcap/go-tpc v0.0.0-20200229030315-98ee0f8f09d3
	github.com/pingcap/kvproto v0.0.0-20200317095539-c42a1d8db7d3
	github.com/pingcap/log v0.0.0-20200117041106-d28c14d3b1cd

	github.com/pingcap/parser v0.0.0-20200109073933-a9496438d77d
	github.com/pingcap/pd v2.1.17+incompatible
	github.com/pingcap/sysutil v0.0.0-20200309085538-962fd285f3bb // indirect
	github.com/pingcap/tidb v2.1.0-beta+incompatible
	github.com/pingcap/tidb-operator v1.1.0-beta.1.0.20200326133238-0fe67bf5e069
	github.com/rogpeppe/fastuuid v1.2.0

	github.com/satori/go.uuid v1.2.0
	github.com/stretchr/testify v1.5.1
	github.com/tikv/client-go v0.0.0-20200110101306-a3ebdb020c83
	github.com/uber-go/atomic v1.5.0 // indirect
	golang.org/x/net v0.0.0-20200301022130-244492dfa37a
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	golang.org/x/tools v0.0.0-20200301222351-066e0c02454c // indirect
	google.golang.org/genproto v0.0.0-20200117163144-32f20d992d24 // indirect
	gopkg.in/yaml.v2 v2.2.8 // indirect
	k8s.io/api v0.0.0
	k8s.io/apimachinery v0.0.0
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/utils v0.0.0-20191114200735-6ca3b61696b6
	sigs.k8s.io/controller-runtime v0.4.0
)

replace google.golang.org/grpc => google.golang.org/grpc v1.26.0

replace github.com/pingcap/pd => github.com/pingcap/pd v1.1.0-beta.0.20200106144140-f5a7aa985497

replace github.com/pingcap/tidb => github.com/pingcap/tidb v1.1.0-beta.0.20200110034112-1b34cc234e82

replace github.com/prometheus/prometheus => github.com/prometheus/prometheus v1.8.2-0.20200213233353-b90be6f32a33

replace github.com/renstrom/dedent => github.com/lithammer/dedent v1.1.0

replace github.com/uber-go/atomic => go.uber.org/atomic v1.5.0

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

replace github.com/Azure/go-autorest => github.com/Azure/go-autorest v12.2.0+incompatible

replace golang.org/x/net v0.0.0-20190813000000-74dc4d7220e7 => golang.org/x/net v0.0.0-20190813141303-74dc4d7220e7
