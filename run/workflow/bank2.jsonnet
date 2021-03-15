{
  _config+:: {
    case_name: 'bank2',
    image_name: 'hub.pingcap.net/qa/tipocket',
    args+: {
      // k8s configurations
      // 'storage-class': 'local-storage',
      'tikv-replicas': '4',
    },
    command: { concurrency: '200', accounts: '1000000', tidb_replica_read: 'leader-and-follower' },
  },
}
