module github.com/pingcap/tipocket

go 1.13

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/DATA-DOG/go-sqlmock v1.4.1 // indirect
	github.com/ScaleFT/sshkeys v0.0.0-20200327173127-6142f742bca5 // indirect
	github.com/alecthomas/assert v0.0.0-20170929043011-405dbfeb8e38
	github.com/alecthomas/colour v0.1.0 // indirect
	github.com/alecthomas/repr v0.0.0-20200325044227-4184120f674c // indirect
	github.com/anishathalye/porcupine v0.0.0-20200229220004-848b8b5d43d9
	github.com/appleboy/easyssh-proxy v1.3.7
	github.com/chaos-mesh/matrix v0.0.0-20200715113735-688b14661cd8
	github.com/codahale/hdrhistogram v0.0.0-20161010025455-3a0bb77429bd
	github.com/cznic/mathutil v0.0.0-20181122101859-297441e03548
	github.com/docker/docker v0.7.3-0.20190817195342-4760db040282
	github.com/gengliqi/persistent_treap v0.0.0-20200403155416-2b2a1532211c
	github.com/go-sql-driver/mysql v1.5.0
	github.com/goccy/go-graphviz v0.0.5
	github.com/google/uuid v1.1.1
	github.com/gorilla/mux v1.8.0
	github.com/grafana/loki v1.3.1-0.20200316172301-1eb139c37c1c
	github.com/jinzhu/gorm v1.9.16
	github.com/json-iterator/go v1.1.10 // indirect
	github.com/juju/errors v0.0.0-20190930114154-d42613fe1ab9
	github.com/mgechev/revive v1.0.2
	github.com/mohae/deepcopy v0.0.0-20170603005431-491d3605edfb
	github.com/ngaut/log v0.0.0-20180314031856-b8e36e7ba5ac
	github.com/pingcap/advanced-statefulset v0.3.2
	github.com/pingcap/chaos-mesh v0.0.0-20200519114056-7201f6b64797
	github.com/pingcap/errors v0.11.5-0.20190809092503-95897b64e011
	github.com/pingcap/go-tpc v0.0.0-20200229030315-98ee0f8f09d3
	github.com/pingcap/kvproto v0.0.0-20200324130106-b8bc94dd8a36
	github.com/pingcap/log v0.0.0-20200511115504-543df19646ad
	github.com/pingcap/parser v0.0.0-20200317021010-cd90cc2a7d87
	github.com/pingcap/pd v2.1.17+incompatible
	github.com/pingcap/pd/v4 v4.0.0-beta.1.0.20200305072537-61d9f9cc35d3
	github.com/pingcap/tidb v2.1.0-beta+incompatible
	github.com/pingcap/tidb-operator v1.1.0-rc.3
	github.com/pingcap/tidb-tools v4.0.1-0.20200612040216-6ddacc75561c+incompatible
	github.com/pkg/errors v0.9.1
	github.com/rogpeppe/fastuuid v1.2.0
	github.com/satori/go.uuid v1.2.0
	github.com/sergi/go-diff v1.1.0 // indirect
	github.com/spf13/cobra v0.0.6
	github.com/stretchr/testify v1.5.1
	github.com/tikv/client-go v0.0.0-20200110101306-a3ebdb020c83
	github.com/uber-go/atomic v1.5.0
	go.uber.org/zap v1.14.0
	golang.org/x/crypto v0.0.0-20200820211705-5c72a883971a // indirect
	golang.org/x/net v0.0.0-20200520004742-59133d7f0dd7
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	golang.org/x/sys v0.0.0-20200826173525-f9321e4c35a6 // indirect
	google.golang.org/protobuf v1.25.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
	k8s.io/api v0.17.0
	k8s.io/apimachinery v0.17.0
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/utils v0.0.0-20191114200735-6ca3b61696b6
	sigs.k8s.io/controller-runtime v0.4.0
)

replace google.golang.org/grpc => google.golang.org/grpc v1.26.0

// we use pingcap/pd and pingcap/pd/v4 at the same time, which will cause a panic because pd register prometheus metrics two times.
replace github.com/pingcap/pd => github.com/mahjonp/pd v1.1.0-beta.0.20200408110858-9c088a87390c

replace github.com/pingcap/tidb => github.com/pingcap/tidb v0.0.0-20200317142013-5268094afe05

replace github.com/prometheus/prometheus => github.com/prometheus/prometheus v1.8.2-0.20200213233353-b90be6f32a33

replace github.com/uber-go/atomic => go.uber.org/atomic v1.5.0

replace k8s.io/api => k8s.io/api v0.17.0

replace k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.17.0

replace k8s.io/apimachinery => k8s.io/apimachinery v0.17.0

replace k8s.io/apiserver => k8s.io/apiserver v0.17.0

replace k8s.io/cli-runtime => k8s.io/cli-runtime v0.0.0-20190918162238-f783a3654da8

replace k8s.io/client-go => k8s.io/client-go v0.0.0-20190918160344-1fbdaa4c8d90

replace k8s.io/code-generator => k8s.io/code-generator v0.0.0-20190912054826-cd179ad6a269

replace k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.0.0-20190918161219-8c8f079fddc3

replace k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.0.0-20190918162944-7a93a0ddadd8

replace k8s.io/kube-proxy => k8s.io/kube-proxy v0.0.0-20190918162534-de037b596c1e

replace k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.0.0-20190918162820-3b5c1246eb18

replace k8s.io/kubelet => k8s.io/kubelet v0.0.0-20190918162654-250a1838aa2c

replace k8s.io/metrics => k8s.io/metrics v0.0.0-20190918162108-227c654b2546

replace k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.0.0-20190918161442-d4c9c65c82af

replace k8s.io/cloud-provider => k8s.io/cloud-provider v0.0.0-20190918163234-a9c1f33e9fb9

replace k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.0.0-20190918163108-da9fdfce26bb

replace k8s.io/component-base => k8s.io/component-base v0.0.0-20190918160511-547f6c5d7090

replace k8s.io/cri-api => k8s.io/cri-api v0.0.0-20190828162817-608eb1dad4ac

replace k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.0.0-20190918163402-db86a8c7bb21

replace k8s.io/kubectl => k8s.io/kubectl v0.0.0-20190918164019-21692a0861df

replace k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.0.0-20190918163543-cfa506e53441

replace github.com/Azure/go-autorest => github.com/Azure/go-autorest v12.2.0+incompatible

replace golang.org/x/net v0.0.0-20190813000000-74dc4d7220e7 => golang.org/x/net v0.0.0-20190813141303-74dc4d7220e7
