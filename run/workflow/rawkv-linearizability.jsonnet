{
  _config+:: {
    case_name: 'rawkv_linearizability',
    image_name: 'hub.pingcap.net/qa/tipocket',
    args+: {
      // k8s configurations
      // 'storage-class': 'local-storage',
      // client configurations
      client: 5,
      'request-count': 20000,
      round: 50,
    },
    command: {},
  },
}
