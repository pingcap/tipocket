{
  _config+:: {
    case_name: 'block-writer',
    image_name: 'hub.pingcap.net/qa/tipocket',
    args+: {
      // k8s configurations
      // 'storage-class': 'local-storage',
    },
    command: {},
  },
}
