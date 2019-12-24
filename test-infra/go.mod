module github.com/pingcap/tipocket/test-infra

go 1.13

require (
	github.com/Nvveen/Gotty v0.0.0-20120604004816-cd527374f1e5 // indirect
	github.com/anishathalye/porcupine v0.0.0-20190205033716-f6fec466e840
	github.com/coreos/bbolt v1.3.3 // indirect
	github.com/coreos/etcd v3.3.17+incompatible // indirect
	github.com/coreos/gofail v0.0.0-20190801230047-ad7f989257ca // indirect
	github.com/cznic/sortutil v0.0.0-20181122101858-f5f958428db8 // indirect
	github.com/daviddengcn/go-colortext v0.0.0-20180409174941-186a3d44e920 // indirect
	github.com/dnephin/govet v0.0.0-20171012192244-4a96d43e39d3 // indirect
	github.com/go-sql-driver/mysql v1.4.1
	github.com/juju/errors v0.0.0-20190930114154-d42613fe1ab9
	github.com/myesui/uuid v1.0.0 // indirect
	github.com/onsi/ginkgo v1.10.3
	github.com/onsi/gomega v1.7.1
	github.com/pingcap/advanced-statefulset v0.2.2
	github.com/pingcap/chaos-mesh v0.0.0-20191224064238-48f8c08a450c
	github.com/pingcap/errors v0.11.5-0.20190809092503-95897b64e011
	github.com/pingcap/gofail v0.0.0-20181217135706-6a951c1e42c3 // indirect
	github.com/pingcap/goleveldb v0.0.0-20191031114657-7683883cfb36 // indirect
	github.com/pingcap/tidb v2.1.0-beta+incompatible
	github.com/pingcap/tidb-operator v1.1.0-alpha.4.0.20191224115938-02e8501761a5
	github.com/pingcap/tipb v0.0.0-20191209145133-44f75c9bef33 // indirect
	github.com/pingcap/tipocket v0.0.0-20191213034629-847195f429d0 // indirect
	github.com/pingcap/tipocket/pocket v0.0.0-20191218112423-aebd26969052
	github.com/spaolacci/murmur3 v1.1.0 // indirect
	github.com/twinj/uuid v1.0.0 // indirect
	github.com/uber-go/atomic v1.4.0 // indirect
	go.etcd.io/etcd v0.5.0-alpha.5.0.20190320044326-77d4b742cdbf
	golang.org/x/sys v0.0.0-20191210023423-ac6580df4449 // indirect
	golang.org/x/tools v0.0.0-20191213032237-7093a17b0467 // indirect
	google.golang.org/appengine v1.6.5 // indirect
	gopkg.in/square/go-jose.v2 v2.3.0 // indirect
	k8s.io/api v0.0.0
	k8s.io/apimachinery v0.0.0
	k8s.io/client-go v0.0.0
	k8s.io/component-base v0.0.0
	k8s.io/csi-api v0.0.0-00010101000000-000000000000 // indirect
	k8s.io/klog v1.0.0
	k8s.io/kubernetes v1.16.0
	k8s.io/node-api v0.0.0-00010101000000-000000000000 // indirect
	k8s.io/utils v0.0.0-20191114184206-e782cd3c129f
	sigs.k8s.io/controller-runtime v0.4.0
	vbom.ml/util v0.0.0-20180919145318-efcd4e0f9787 // indirect
)

replace github.com/pingcap/tidb => github.com/you06/tidb v1.1.0-beta.0.20191107083526-0edcbcc52610

replace github.com/pingcap/tipocket/go-sqlsmith => github.com/you06/tipocket/go-sqlsmith v0.0.0-20191209063433-0cdfd1ff9a02

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
