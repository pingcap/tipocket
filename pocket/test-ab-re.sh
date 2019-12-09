#!/bin/bash

rm test-log/log/reproduce.log


# ./bin/doppelganger -concurrency 6 -dsn1 "root:@tcp(172.17.0.1:4000)/sqlsmith" -dsn2 "root@tcp(172.16.5.115:4000)/sqlsmith" -re ./test-log/log
# ./bin/doppelganger -concurrency 6 -dsn1 "root:@tcp(172.17.0.1:4000)/sqlsmith" -dsn2 "root@tcp(172.17.0.1:14000)/sqlsmith" -re ./test-log/log
# ./bin/doppelganger -concurrency 6 -dsn1 "root:@tcp(172.17.0.1:4000)/sqlsmith" -dsn2 "root@tcp(172.16.5.206:14000)/sqlsmith" -re ./test-log/log-abtest
./bin/doppelganger -concurrency 6 -dsn1 "root@tcp(172.17.0.1:33306)/sqlsmith" -dsn2 "root:@tcp(172.17.0.1:4000)/sqlsmith" -re ./test-log/log-abtest
