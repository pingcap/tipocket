#!/usr/bin/env bash

set -ex
env

echo "get resource info"
curl -v "${API_SERVER}"/api/cluster/resource/"${CLUSTER_ID}"

echo "scale_out cluster"
curl -v "${API_SERVER}"/api/cluster/scale_out/"${CLUSTER_ID}"/4/tikv

echo "upload report"
curl -v "${API_SERVER}"/api/cluster/workload/"${CLUSTER_ID}"/result -H "Content-Type: application/json" -X POST -d '{"plaintext": "cluster testing..."}'

echo "get report"
curl -v "${API_SERVER}"/api/cluster/workload/"${CLUSTER_ID}"/result -H "Content-Type: application/json" -X GET

echo "clean cluster data"
curl -v "${API_SERVER}"/api/cluster/clean/"${CLUSTER_ID}"

echo "check cluster status"
curl -v "${API_SERVER}"/api/cluster/"${CLUSTER_ID}"

while [ $(curl "${API_SERVER}/api/cluster/${CLUSTER_ID}" | jq --exit-status '.status' --raw-output) != "RUNNING" ]; do
    echo "sleep..."
    sleep 30
done

echo "Pass"

