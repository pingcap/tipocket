{
  _config:: {
    image_name: 'hub.pingcap.net/qa/tipocket',
    args: {
      // cluster configurations
      hub: 'docker.io',
      repository: 'pingcap',
      image_version: 'nightly',
      tidb_image: '',
      tikv_image: '',
      pd_image: '',
      tidb_config: '',
      tikv_config: '',
      pd_config: '',
      nemesis: 'random_kill,kill_pd_leader_5min,partition_one,subcritical_skews,big_skews,shuffle-leader-scheduler,shuffle-region-scheduler,random-merge-scheduler',
      // k8s configurations
      namespace: '{{workflow.name}}',
      storage_class: 'local-path',
      purge: 'false',
      delete_ns: 'false',
      // client configurations
      client: 1,
      request_count: 10000,
      round: 1,
    },
  },
}
