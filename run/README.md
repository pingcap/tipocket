# Intro

This directory contains some manifests for running TiPocket in [argo](https://github.com/argoproj/argo).

Don't worry, usually, you're able to run the cases quickly even you don't know the argo workflow very well.

Here we use jsonnet to generate workflows based on some basic metadata. So for test case writers, you only need to fill in very little metadata letting us know how to generate the workflow.

## Usage

For example, If I have written a test case named `list-append` which has three parameters: tablecount, read_lock, txn_mode.

To let it runnable, I need to fill on the metadata about list-append case on [lib/case.jsonnet](./lib/case.jsonnet) just like below:

```jsonnet
list_append(args={ tablecount: '7', read_lock: '"FOR UPDATE"', txn_mode: 'pessimistic' })::
  [
    '/bin/append',
    '-table-count=%s' % args.tablecount,
    '-read-lock=%s' % args.read_lock,
    '-txn-mode=%s' % args.txn_mode,
  ],
```

Looks simple right? Jsonnet is a simple extension of JSON, on above I define a function named list_append which has three arguments, and returns an array.

After fill on the testcase metadata, I should write a workflow file on [workflow/list-append.jsonnet](./workflow/list-append.jsonnet), it's content:

```jsonnet
{
  _config+:: {
    case_name: 'list_append',
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
```

Explain a little here, `_config+` means override some key-value defined on [lib/config.jsonnet](./lib/config.jsonnet).

On the `args+` part, I set `storage-class`, `client` and `round` etc parameters declared on the `fixture.go` file which is the common flag all over test cases.

Then, the `command` field is set to invoke list_append case with 7 tables, use `FOR UPDATE` as the read_lock.

Now I could run `make build_workflow` to generate the workflow file.

```bash
$ make build_workflow
find run -name "*.jsonnet" | xargs -I{} jsonnetfmt -i {}
rm -rf run/manifest/*
find run/workflow -name "*.jsonnet" -type f -exec basename {} \;  | xargs -I% sh -c 'jsonnet run/workflow/% -J run/lib | yq eval -P - > run/manifest/%.yaml'
```

Lucky, everything goes well. ☺

## Run a case

Since we already have the workflow manifest, it is more recommended to use argo-cli to submit the workflow.

For downloading argo-cli, please check [releases](https://github.com/argoproj/argo/releases).

After downloading the cli, we can list the current workflows, submit a new workflow.

### List all workflows

``` bash
argo list -n argo

NAME                                  STATUS      AGE   DURATION   PRIORITY
tipocket-bank-timechaos-tfzvk         Running     8m    8m         0
tipocket-bank-timechaos-46ng9         Failed      11m   55s        0
tipocket-ledger-iochaos-gbf9m         Succeeded   1h    1h         0
tipocket-ledger-iochaos-wx4zl         Succeeded   2h    1h         0
```

### Submit a new workflow

```bash
 argo submit run/manifest/workflow/vbank.jsonnet.yaml -n argo
Name:                tipocket-vbank-9whq7
Namespace:           argo
ServiceAccount:      default
Status:              Pending
Created:             Thu Feb 04 11:28:52 +0800 (now)
```

### Validate workflow manifests

```bash
$ argo lint run/manifest/workflow/*.yaml
run/manifest/workflow/bank.jsonnet.yaml is valid
run/manifest/workflow/bank2.jsonnet.yaml is valid
run/manifest/workflow/block-writer.jsonnet.yaml is valid
run/manifest/workflow/cross-region.jsonnet.yaml is valid
run/manifest/workflow/example.jsonnet.yaml is valid
run/manifest/workflow/ledger.jsonnet.yaml is valid
run/manifest/workflow/list-append.jsonnet.yaml is valid
run/manifest/workflow/rawkv-linearizability.jsonnet.yaml is valid
run/manifest/workflow/region-available.jsonnet.yaml is valid
run/manifest/workflow/rw-register.jsonnet.yaml is valid
run/manifest/workflow/sqllogic.jsonnet.yaml is valid
run/manifest/workflow/tpcc.jsonnet.yaml is valid
run/manifest/workflow/txn-rand-pessimistic.jsonnet.yaml is valid
run/manifest/workflow/vbank.jsonnet.yaml is valid
Workflow manifests validated
```
