(import 'argo/workflow.jsonnet') +
(import 'case.jsonnet') +
(import 'util.jsonnet') +
(import 'config.jsonnet') +
{
  _config+:: {
    case_name: 'list-append',
    image_name: 'hub.pingcap.net/qa/tipocket',
    args+: {
    },
    command: $.list_append(tablecount='7', read_lock='"FOR UPDATE"', txn_mode='pessimistic'),
  },
}
