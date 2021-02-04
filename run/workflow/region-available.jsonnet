(import 'argo/argo.jsonnet') +
(import 'case.jsonnet') +
(import 'util.jsonnet') +
(import 'config.jsonnet') +
{
  _config+:: {
    case_name: 'region-available',
    image_name: 'hub.pingcap.net/qa/tipocket',
    args+: {
      // k8s configurations
      'storage-class': 'local-storage',
      // client configurations
      'run-time': '4h',
      round: 50,
    },
    command: $.region_available(),
  },
}
