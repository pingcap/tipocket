{
  'list-append'(args={ tablecount: '7', read_lock: '"FOR UPDATE"', txn_mode: 'pessimistic' })::
    [
      '/bin/list-append',
      '-table-count=%s' % args.tablecount,
      '-read-lock=%s' % args.read_lock,
      '-txn-mode=%s' % args.txn_mode,
    ],
  'rw-register'(args={ tablecount: '7', read_lock: '"FOR UPDATE"', txn_mode: 'pessimistic' })::
    [
      '/bin/rw-register',
      '-table-count=%s' % args.tablecount,
      '-read-lock=%s' % args.read_lock,
      '-txn-mode=%s' % args.txn_mode,
    ],
  ledger(args={})::
    [
      '/bin/ledger',
    ],
  rawkv_linearizability(args={})::
    [
      '/bin/rawkv-linearizability',
    ],
  region_available(args={})::
    [
      '/bin/region-available',
    ],
  bank(args={})::
    [
      '/bin/bank',
    ],
  'bank2'(args={ concurrency: '200', accounts: '1000000', tidb_replica_read: 'leader-and-follower' })::
    [
      '/bin/bank2',
      '-concurrency=%s' % args.concurrency,
      '-accounts=%s' % args.accounts,
      '-tidb-replica-read=%s' % args.tidb_replica_read,
    ],
  sqllogic(args={})::
    [
      '/bin/sqllogic',
    ],
  tpcc(args={})::
    [
      '/bin/tpcc',
    ],
  txn_rand_pessimistic(args={})::
    [
      '/bin/txn-rand-pessimistic',
    ],
  vbank(args={ connParams: '' })::
    [
      '/bin/vbank',
      '-conn_params=%s' % args.connParams,
    ],
  crossregion(args={ tso_request_count: '2000' })::
    [
      '/bin/cross-region',
       '-tso-request-count=%s' % args.tso_request_count,
    ],
  'example'(args={})::
    [
      '/bin/example',
    ],
  'block-writer'(args={})::
    [
      '/bin/block-writer',
    ],
  'stale-read'(args={})::
    [
      '/bin/stale-read',
    ],
  'ttl'(args={})::
    [
      '/bin/tll',
    ],
  // +tipocket:scaffold:case_decls
}
