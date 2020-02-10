package cluster

import (
	"context"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/pingcap/tipocket/pkg/test-infra/pkg/tidb"
	"github.com/pingcap/tipocket/pkg/test-infra/tests/util"
	corev1 "k8s.io/api/core/v1"

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
func (k *K8sProvisioner) SetUp(ctx context.Context, spec interface{}) ([]Node, error) {
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

func (k *K8sProvisioner) setUpTiDBCluster(ctx context.Context, recommand *tidb.TiDBClusterRecommendation) ([]Node, error) {
	var (
		nodes []Node
		err   error
	)

	err = k.E2eCli.TiDB.ApplyTiDBCluster(recommand.TidbCluster)
	if err != nil {
		return nodes, err
	}

	// TODO: use ctx for wait end
	err = k.E2eCli.TiDB.WaitTiDBClusterReady(recommand.TidbCluster, 10*time.Minute)
	if err != nil {
		return nodes, err
	}

	err = k.E2eCli.TiDB.ApplyTiDBService(recommand.Service)
	if err != nil {
		return nodes, err
	}

	pods, err := k.E2eCli.TiDB.GetNodes(recommand.TidbCluster.ObjectMeta.Namespace)
	if err != nil {
		return nodes, err
	}
	nodes = parseNodeFromPodList(pods)

	k8sNodes, err := k.E2eCli.GetNodes()
	if err != nil {
		return nodes, err
	}
	nodeIP := getNodeIP(k8sNodes)

	svc, err := k.E2eCli.TiDB.GetTiDBServiceByMeta(&recommand.Service.ObjectMeta)
	for i, node := range nodes {
		if tidbRegex.MatchString(node.PodName) {
			node.IP = nodeIP
			node.Port = getTiDBNodePort(svc)
		}
		nodes[i] = node
	}

	log.Println(nodes)

	return nodes, err
}

func parseNodeFromPodList(pods *corev1.PodList) []Node {
	var nodes []Node
	for _, pod := range pods.Items {
		if strings.Contains(pod.ObjectMeta.Name, "discovery") {
			continue
		}
		nodes = append(nodes, Node{
			Namespace: pod.ObjectMeta.Namespace,
			PodName:   pod.ObjectMeta.Name,
			IP:        pod.Status.PodIP,
			Port:      findPort(pod.ObjectMeta.Name, pod.Spec.Containers[0].Ports),
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
