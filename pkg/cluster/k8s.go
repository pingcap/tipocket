package cluster

import (
	"context"
	"os"
	"regexp"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/pingcap/tipocket/pkg/test-infra/pkg/fixture"
	"github.com/pingcap/tipocket/pkg/test-infra/pkg/tidb"
	"github.com/pingcap/tipocket/pkg/test-infra/tests/util"

	"k8s.io/client-go/tools/clientcmd"
)

var (
	tidbRegex = regexp.MustCompile(`.*tidb-[0-9]+$`)
)

// K8sProvisioner implement Provisioner in k8s
type K8sProvisioner struct {
	*util.E2eCli
}

// NewK8sProvisioner create k8s provisioner
func NewK8sProvisioner() (Provisioner, error) {
	conf, err := clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	if err != nil {
		return nil, err
	}
	return &K8sProvisioner{
		E2eCli: util.NewE2eCli(conf),
	}, nil
}

// SetUp sets up cluster, returns err or all nodes info
func (k *K8sProvisioner) SetUp(ctx context.Context, spec interface{}) ([]Node, []ClientNode, error) {
	switch s := spec.(type) {
	case *tidb.TiDBClusterRecommendation:
		return k.setUpTiDBCluster(ctx, s)
	default:
		panic("unreachable")
	}
}

// TearDown tears down the cluster
func (k *K8sProvisioner) TearDown() error {
	return nil
}

func (k *K8sProvisioner) setUpTiDBCluster(ctx context.Context, recommend *tidb.TiDBClusterRecommendation) ([]Node, []ClientNode, error) {
	var (
		nodes       []Node
		clientNodes []ClientNode
		err         error
	)
	err = k.E2eCli.TiDB.ApplyTiDBCluster(recommend.TidbCluster)
	if err != nil {
		return nodes, clientNodes, err
	}

	// TODO: use ctx for wait end
	err = k.E2eCli.TiDB.WaitTiDBClusterReady(recommend.TidbCluster, 10*time.Minute)
	if err != nil {
		return nodes, clientNodes, err
	}

	err = k.E2eCli.TiDB.ApplyTiDBService(recommend.Service)
	if err != nil {
		return nodes, clientNodes, err
	}

	pods, err := k.E2eCli.TiDB.GetNodes(recommend.TidbCluster.ObjectMeta.Namespace)
	if err != nil {
		return nodes, clientNodes, err
	}
	nodes = parseNodeFromPodList(pods)

	// attach the client to every nodes
	for idx := range nodes {
		nodes[idx].Client = &Client{
			Namespace: nodes[idx].Namespace,
			PDMemberFunc: func(ns string) (string, []string, error) {
				return k.TiDB.GetPDMember(ns, ns)
			},
		}
	}
	k8sNodes, err := k.E2eCli.GetNodes()
	if err != nil {
		return nodes, clientNodes, err
	}

	svc, err := k.E2eCli.TiDB.GetTiDBServiceByMeta(&recommend.Service.ObjectMeta)
	if err != nil {
		return nodes, clientNodes, err
	}
	clientNodes = append(clientNodes, ClientNode{
		Namespace: svc.ObjectMeta.Namespace,
		Component: TiDB,
		IP:        getNodeIP(k8sNodes),
		Port:      getTiDBNodePort(svc),
	})

	return nodes, clientNodes, err
}

func parseNodeFromPodList(pods *corev1.PodList) []Node {
	var nodes []Node
	for _, pod := range pods.Items {
		if strings.Contains(pod.ObjectMeta.Name, "discovery") {
			continue
		}
		var component Component
		cmp, ok := pod.Labels["app.kubernetes.io/component"]
		if !ok {
			component = Unknown
		}
		switch cmp {
		case string(TiDB):
			component = TiDB
		case string(TiKV):
			component = TiKV
		case string(PD):
			component = PD
		default:
			component = Unknown
		}
		nodes = append(nodes, Node{
			Namespace: pod.ObjectMeta.Namespace,
			Component: component,
			// TODO use better way to retrieve version?
			Version: fixture.E2eContext.ImageVersion,
			PodName: pod.ObjectMeta.Name,
			IP:      pod.Status.PodIP,
			Port:    findPort(pod.ObjectMeta.Name, pod.Spec.Containers[0].Ports),
		})
	}
	return nodes
}

func findPort(podName string, ports []corev1.ContainerPort) int32 {
	if len(ports) == 0 {
		return 0
	}

	var priorityPort int32 = 0
	if strings.Contains(podName, "pd") {
		priorityPort = 2379
	} else if strings.Contains(podName, "tikv") {
		priorityPort = 20160
	} else if strings.Contains(podName, "tidb") {
		priorityPort = 4000
	}

	for _, port := range ports {
		if port.ContainerPort == priorityPort {
			return priorityPort
		}
	}

	return ports[0].ContainerPort
}

func getTiDBNodePort(svc *corev1.Service) int32 {
	for _, port := range svc.Spec.Ports {
		if port.Port == 4000 {
			return port.NodePort
		}
	}
	return 0
}

func getNodeIP(nodeList *corev1.NodeList) string {
	if len(nodeList.Items) == 0 {
		return ""
	}
	return nodeList.Items[0].Status.Addresses[0].Address
}
