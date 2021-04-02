{
  _config+:: {
    case_name: 'tpcc',
    image_name: 'hub.pingcap.net/tipocket/tipocket',
    args+: {
      // k8s configurations
      // 'storage-class': 'local-storage',
      'tikv-replicas': '4',
    },
    command: {},
  },
}
