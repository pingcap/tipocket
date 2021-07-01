{
  _config+:: {
    case_name: 'list-append',
    image_name: 'hub.pingcap.net/qa/tipocket',
    args+: {
      // k8s configurations
      // 'storage-class': 'local-storage',
      // client configurations
      client: 5,
      'request-count': 100000,
      round: 10,
    },
    command: { tablecount: '7', read_lock: '"FOR UPDATE"', txn_mode: 'pessimistic' },
  },
}
