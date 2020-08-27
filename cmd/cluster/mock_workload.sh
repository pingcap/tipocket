#!/usr/bin/env bash

set -ex
env

curl -v "${API_SERVER}"/api/cluster/resource/"${CLUSTER_NAME}"

curl -v "${API_SERVER}"/api/cluster/scale_out/"${CLUSTER_NAME}"/4/tikv

curl -v 172.16.5.110:8000/api/cluster/workload/"$CLUSTER_NAME"/result -H "Content-Type: application/json" -X GET
curl -v 172.16.5.110:8000/api/cluster/workload/"$CLUSTER_NAME"/result -H "Content-Type: application/json" -X POST -d '{"plaintext": "this is result..."}'