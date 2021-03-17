{
  _config:: {
    schedule: '0 0 * * *',
    image_name: 'hub.pingcap.net/qa/tipocket',
    args: {
      // cluster configurations
      hub: 'docker.io',
      repository: 'pingcap',
      'image-version': 'nightly',
      'tidb-image': '',
      'tikv-image': '',
      'pd-image': '',
      'tidb-config': '',
      'tikv-config': '',
      'pd-config': '',
      'tikv-replicas': '3',
      'prepare-sql': '',
      nemesis: 'random_kill,kill_pd_leader_5min,partition_one,subcritical_skews,big_skews,shuffle-leader-scheduler,shuffle-region-scheduler,random-merge-scheduler',
      'failpoint.tidb': '',
      // k8s configurations
      namespace: '{{workflow.name}}',
      'cluster-name': '{{workflow.name}}',
      'storage-class': 'local-path',
      purge: 'false',
      delNS: 'false',
      // client configurations
      client: 1,
      'request-count': 10000,
      'run-time': '100m',
      round: 1,
    },
  },
}
