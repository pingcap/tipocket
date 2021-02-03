{
  fixture(namespace, purge="false", hub="docker.io", repository="pingcap", 
  image_version="nightly", tidb_image="", tikv_image="", pd_image="", 
  storage_class="local-path", nemesis="random_kill,kill_pd_leader_5min,partition_one,subcritical_skews,big_skews,shuffle-leader-scheduler,shuffle-region-scheduler,random-merge-scheduler",
  client=1, request_count=10000,round=1,
  tidb_config="", tikv_config="", pd_config="",
  delete_ns="false")::
    [
      '-namespace=%s' % namespace,
      '-purge=%s' % purge,
      '-hub=%s' % hub,
      '-repository=%s' % repository,
      '-image-version=%s' % image_version,
      '-tidb-image=%s' % tidb_image,
      '-tikv-image=%s' % tikv_image,
      '-pd-image=%s' % pd_image,
      '-storage-class=%s' % storage_class,
      '-nemesis=%s' % nemesis,
      '-client=%d' % client,
      '-request-count=%d' % request_count,
      '-round=%d' % round,
      '-tidb-config=%s' % tidb_config,
      '-tikv-config=%s' % tikv_config,
      '-pd-config=%s' % pd_config,
      '-delNS=%s' % delete_ns,
    ],
  list_append(tablecount, read_lock, txn_mode)::
    [
      '/bin/append',
      '-table-count=%s' % tablecount,
      '-read-lock=%s' % read_lock,
      '-txn-mode=%s' % txn_mode,
    ],
}
