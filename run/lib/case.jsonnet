{
  list_append(tablecount, read_lock, txn_mode)::
    [
      '/bin/append',
      '-table-count=%s' % tablecount,
      '-read-lock=%s' % read_lock,
      '-txn-mode=%s' % txn_mode,
    ],
}
