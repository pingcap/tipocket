module github.com/pingcap/tipocket

go 1.13

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/anishathalye/porcupine v0.0.0-20200229220004-848b8b5d43d9
	github.com/chaos-mesh/chaos-mesh/api/v1alpha1 v0.0.0-20210118085126-27d3a05f8242
	github.com/chaos-mesh/matrix v0.0.0-20200715113735-688b14661cd8
	github.com/codahale/hdrhistogram v0.0.0-20161010025455-3a0bb77429bd
	github.com/cznic/mathutil v0.0.0-20181122101859-297441e03548
	github.com/gengliqi/persistent_treap v0.0.0-20200403155416-2b2a1532211c
	github.com/go-sql-driver/mysql v1.5.0
	github.com/goccy/go-graphviz v0.0.5
	github.com/google/go-cmp v0.5.0
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/uuid v1.1.1
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/grafana/loki v1.3.1-0.20200316172301-1eb139c37c1c
	github.com/hashicorp/go-multierror v1.1.0 // indirect
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/json-iterator/go v1.1.10 // indirect
	github.com/juju/errors v0.0.0-20190930114154-d42613fe1ab9
	github.com/juju/loggo v0.0.0-20180524022052-584905176618 // indirect
	github.com/juju/testing v0.0.0-20180920084828-472a3e8b2073 // indirect
	github.com/klauspost/cpuid v1.3.1 // indirect
	github.com/mattn/go-colorable v0.1.7 // indirect
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/mgechev/revive v1.0.2
	github.com/mohae/deepcopy v0.0.0-20170603005431-491d3605edfb
	github.com/ngaut/log v0.0.0-20180314031856-b8e36e7ba5ac
	github.com/onsi/gomega v1.10.1
	github.com/pingcap/advanced-statefulset v0.3.2
	github.com/pingcap/advanced-statefulset/client v1.16.0
	github.com/pingcap/errors v0.11.5-0.20190809092503-95897b64e011
	github.com/pingcap/failpoint v0.0.0-20200702092429-9f69995143ce // indirect
	github.com/pingcap/go-tpc v0.0.0-20200229030315-98ee0f8f09d3
	github.com/pingcap/kvproto v0.0.0-20200324130106-b8bc94dd8a36
	github.com/pingcap/parser v0.0.0-20200317021010-cd90cc2a7d87
	github.com/pingcap/pd v2.1.17+incompatible
	github.com/pingcap/tidb v2.1.0-beta+incompatible
	github.com/pingcap/tidb-tools v4.0.1-0.20200612040216-6ddacc75561c+incompatible // indirect
	github.com/prometheus/prometheus v1.8.2 // indirect
	github.com/rogpeppe/fastuuid v1.2.0
	github.com/stretchr/testify v1.5.1
	github.com/tikv/client-go v0.0.0-20200110101306-a3ebdb020c83
	github.com/uber-go/atomic v1.5.0
	golang.org/x/crypto v0.0.0-20200820211705-5c72a883971a // indirect
	golang.org/x/net v0.0.0-20200904194848-62affa334b73
	golang.org/x/sync v0.0.0-20200625203802-6e8e738ad208
	golang.org/x/sys v0.0.0-20200918174421-af09f7315aff // indirect
	golang.org/x/text v0.3.3 // indirect
	golang.org/x/tools v0.0.0-20200921210052-fa0125251cc4 // indirect
	google.golang.org/protobuf v1.25.0 // indirect
	gopkg.in/mgo.v2 v2.0.0-20180705113604-9856a29383ce // indirect
	k8s.io/api v0.17.0
	k8s.io/apiextensions-apiserver v0.17.0
	k8s.io/apimachinery v0.17.0
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/kubernetes v1.17.0 // indirect
	k8s.io/utils v0.0.0-20200912215256-4140de9c8800
	sigs.k8s.io/controller-runtime v0.4.0
)

replace google.golang.org/grpc => google.golang.org/grpc v1.26.0

// we use pingcap/pd and pingcap/pd/v4 at the same time, which will cause a panic because pd register prometheus metrics two times.
replace github.com/pingcap/pd => github.com/mahjonp/pd v1.1.0-beta.0.20200408110858-9c088a87390c

replace github.com/pingcap/tidb => github.com/pingcap/tidb v0.0.0-20200317142013-5268094afe05

replace github.com/prometheus/prometheus => github.com/prometheus/prometheus v1.8.2-0.20200213233353-b90be6f32a33

replace github.com/uber-go/atomic => go.uber.org/atomic v1.5.0

replace (
	k8s.io/api => k8s.io/api v0.17.0
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.17.0
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.0
	k8s.io/apiserver => k8s.io/apiserver v0.17.0
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.17.0
	k8s.io/client-go => k8s.io/client-go v0.17.0
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.17.0
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.17.0
	k8s.io/code-generator => k8s.io/code-generator v0.17.0
	k8s.io/component-base => k8s.io/component-base v0.17.0
	k8s.io/cri-api => k8s.io/cri-api v0.17.0
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.17.0
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.17.0
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.17.0
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.17.0
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.17.0
	k8s.io/kubectl => k8s.io/kubectl v0.17.0
	k8s.io/kubelet => k8s.io/kubelet v0.17.0
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.17.0
	k8s.io/metrics => k8s.io/metrics v0.17.0
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.17.0
)

replace github.com/Azure/go-autorest => github.com/Azure/go-autorest v12.2.0+incompatible

replace golang.org/x/net v0.0.0-20190813000000-74dc4d7220e7 => golang.org/x/net v0.0.0-20190813141303-74dc4d7220e7
