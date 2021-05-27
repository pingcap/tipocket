{
  _config+:: {
    case_name: 'ledger',
    image_name: 'hub.pingcap.net/tipocket/tipocket',
    args+: {
      // k8s configurations
      // 'storage-class': 'local-storage',
    },
    command: {},
  },
}
