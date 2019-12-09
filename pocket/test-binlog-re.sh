#!/bin/bash

rm test-log/log-binlog/reproduce.log
./bin/doppelganger -concurrency 10 -dsn1 "root:@tcp(172.16.4.4:30125)/sqlsmith" -dsn2 "root@tcp(172.16.4.4:32180)/sqlsmith" -re ./test-log/log-binlog -binlog
