{
  _config+:: {
    case_name: 'region_available',
    image_name: 'hub.pingcap.net/qa/tipocket',
    args+: {
      // k8s configurations
      // 'storage-class': 'local-storage',
      // client configurations
      'run-time': '4h',
      round: 50,
    },
    command: {},
  },
}
