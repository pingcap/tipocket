{
  _config+:: {
    case_name: 'txn_rand_pessimistic',
    image_name: 'hub.pingcap.net/qa/tipocket',
    args+: {
      // k8s configurations
      // 'storage-class': 'local-storage',
      'tikv-replicas': '4',
    },
    command: {},
  },
}
