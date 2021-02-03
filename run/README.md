# Intro

This directory contains some manifests for running tipocket in [argo](https://github.com/argoproj/argo). The [template](./template) directory
contains the manifests for argo [workflow-template](https://github.com/argoproj/argo/blob/master/docs/workflow-templates.md) resources. And the [workflow](./workflow) directory contains the manifests for argo workflow resources.

For more information about argo and its workflows, this [doc](https://argoproj.github.io/docs/argo/examples/readme.html) is really a good start.

## Usage

Since we already have the workflow and workflow templates manifest, it is more recommended to use argo-cli to submit the workflow.

For downloading argo-cli, please check [releases](https://github.com/argoproj/argo/releases).

After downloading the cli, we can list the current workflows, submit new workflow or template.

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
$ argo submit bank.yaml -n argo

Name:                tipocket-bank-d8vr4
Namespace:           argo
ServiceAccount:      default
Status:              Pending
Created:             Sat Mar 21 13:24:31 -0400 (1 second ago)
Parameters:
  image-version:     latest
  storage-class:     sas
```

### Validate workflow manifests

```bash
$ argo lint target/*.yaml
target/bank.jsonnet.yaml is valid
target/block-writer.jsonnet.yaml is valid
target/ledger.jsonnet.yaml is valid
target/list-append.jsonnet.yaml is valid
target/rawkv-linearizability.jsonnet.yaml is valid
target/region-available.jsonnet.yaml is valid
target/rw-register.jsonnet.yaml is valid
target/scbank.jsonnet.yaml is valid
target/scbank2.jsonnet.yaml is valid
target/sqllogic.jsonnet.yaml is valid
target/tpcc.jsonnet.yaml is valid
target/txn-rand-pessimistic.jsonnet.yaml is valid
target/vbank.jsonnet.yaml is valid
Workflow manifests validated
```
