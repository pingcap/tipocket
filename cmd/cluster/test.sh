#!/usr/bin/env bash

curl 172.16.5.110:8000/api/cluster/test -XPOST -d '{"cluster_request":{"name":"test","version":"nightly"},"cluster_request_topologies":[{"component":"tidb","deploy_path":"/data1","rri_item_id":1},{"component":"pd","deploy_path":"/data1","rri_item_id":1},{"component":"tikv","deploy_path":"/data1","rri_item_id":2},{"component":"tikv","deploy_path":"/data1","rri_item_id":3},{"component":"prometheus","deploy_path":"/data1","rri_item_id":1},{"component":"grafana","deploy_path":"/data1","rri_item_id":1}],"cluster_workload":{"docker_image":"mahjonp/bench","rri_item_id":1}}'
