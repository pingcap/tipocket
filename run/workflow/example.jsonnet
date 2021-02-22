{
  _config+:: {
    case_name: 'example',
    image_name: 'hub.pingcap.net/qa/tipocket',
    args+: {
      // k8s configurations
      // 'storage-class': 'local-storage',
    },
    command: {},
  },
}
