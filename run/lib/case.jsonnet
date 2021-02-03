{
  list_append(tablecount, read_lock, txn_mode)::
    [
      '/bin/append',
      '-table-count=%s' % tablecount,
      '-read-lock=%s' % read_lock,
      '-txn-mode=%s' % txn_mode,
    ],
  rw_register(tablecount, read_lock, txn_mode)::
    [
      '/bin/register',
      '-table-count=%s' % tablecount,
      '-read-lock=%s' % read_lock,
      '-txn-mode=%s' % txn_mode,
    ],
  bank()::
    [
      '/bin/pbank',
    ],
  block_writer()::
    [
      '/bin/block-writer',
    ],
  ledger()::
    [
      '/bin/ledger',
    ],
  rawkv_linearizability()::
    [
      '/bin/rawkv-linearizability',
    ],
  region_available()::
    [
      '/bin/region-available',
    ],
  scbank()::
    [
      '/bin/bank',
    ],
  scbank2(concurrency, accounts, tidb_replica_read)::
    [
      '/bin/bank2',
      '-concurrency=%s' % concurrency,
      '-accounts=%s' % accounts,
      '-tidb-replica-read=%s' % tidb_replica_read,
    ],
  sqllogic()::
    [
      '/bin/sqllogic',
    ],
  tpcc()::
    [
      '/bin/tpcc',
    ],
  txn_rand_pessimistic()::
    [
      '/bin/txn-rand-pessimistic',
    ],
  vbank()::
    [
      '/bin/vbank',
    ],
}
