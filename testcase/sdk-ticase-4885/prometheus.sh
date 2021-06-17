# 80pct Duration
http -p HBb :9090/api/v1/query_range query=="histogram_quantile(0.80, sum(rate(tidb_server_handle_query_duration_seconds_bucket{tidb_cluster=\"testbed-tidbcluster-prometheus-pnenkqwj-tc-mcwygwwl\"}[1m])) by (le))" start==1623949000 end==1623949500 step==5m
# QPS
http -p HBb :9090/api/v1/query_range query=="sum(rate(tidb_executor_statement_total{tidb_cluster=\"testbed-tidbcluster-prometheus-pnenkqwj-tc-mcwygwwl\"}[1m]))" start==1623952537 end==1623952907 step==10s