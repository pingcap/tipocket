{
  _config+:: {
    case_name: 'crossregion',
    image_name: 'hub.pingcap.net/qa/tipocket',
    args+: {
      // k8s configurations
      // 'storage-class': 'local-storage',
      // tidbcluster configurations
      // 'pd-storage-class': 'shared-local-storage',
      // 'tikv-storage-class': 'local-storage',
      // 'log-storage-class': 'shared-sas',
    },
    command: { tso_request_count: '100' },
  },
}
