# Intro

This directory contains some manifests for running tipocket in [argo](https://github.com/argoproj/argo). The [template](./template) directory
contains the manifests for argo [workflow-template](https://github.com/argoproj/argo/blob/master/docs/workflow-templates.md) resources. And the 
[workflow](./workflow) directory contains the manifests for argo workflow resources.

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
argo submit workflow/bank.yaml -n argo

Name:                tipocket-bank-d8vr4
Namespace:           argo
ServiceAccount:      default
Status:              Pending
Created:             Sat Mar 21 13:24:31 -0400 (1 second ago)
Parameters:          
  image_version:     latest
  storage_class:     pd-ssd
```

### Submit a new workflow with parameter overwritten

```bash
argo submit workflow/bank.yaml -p image_version=nightly -n argo

Name:                tipocket-bank-xqnm2
Namespace:           argo
ServiceAccount:      default
Status:              Pending
Created:             Sat Mar 21 13:32:55 -0400 (now)
Parameters:          
  image_version:     nightly
  storage_class:     pd-ssd
```

### Submit a new template

```bash
argo template create template/ledger.yaml -n argo
Name:                tipocket-ledger
Namespace:           argo
Created:             Sat Mar 21 13:21:13 -0400 (now)
```

### Validate workflow manifests

```bash
argo lint workflow/*.yaml

workflow/abtest.yaml is valid
workflow/bank.yaml is valid
workflow/block-writer.yaml is valid
workflow/ledger.yaml is valid
workflow/region-available.yaml is valid
workflow/sqllogic.yaml is valid
workflow/tpcc.yaml is valid
workflow/txn-rand-pessimistic.yaml is valid
Workflow manifests validated
```