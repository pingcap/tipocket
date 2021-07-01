{
  _config+:: {
    case_name: 'stale-read',
    image_name: 'hub.pingcap.net/qa/tipocket',
    args+: {
      // k8s configurations
      // 'storage-class': 'local-storage',
      nemesis: 'delay,loss,kill_tikv_1node_5min,short_kill_tikv_1node,random-merge-scheduler,shuffle-leader-scheduler,shuffle-region-scheduler,partition_one',
    },
    command: {},
  },
}
