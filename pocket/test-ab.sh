#!/bin/bash

rm test-log/common/abtest-log.log
rm -rf test-log/log-abtest
mkdir test-log/log-abtest

nohup ./bin/doppelganger -concurrency 6 -dsn1 "root@tcp(172.17.0.1:33306)/sqlsmith" -dsn2 "root:@tcp(172.17.0.1:4000)/sqlsmith" -log ./test-log/log-abtest -stable >> test-log/common/abtest-log.log  2>& 1  &
