(import 'argo/argo.jsonnet') +
(import 'case.jsonnet') +
(import 'util.jsonnet') +
(import 'config.jsonnet') +
{
  _config+:: {
    case_name: 'scbank2',
    image_name: 'hub.pingcap.net/qa/tipocket',
    args+: {
      // k8s configurations
      'storage-class': 'local-storage',
      'tikv-replicas': '4',
    },
    command: $.scbank2(concurrency='200', accounts='1000000', tidb_replica_read='leader-and-follower'),
  },
}
