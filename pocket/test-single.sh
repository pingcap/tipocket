#!/bin/bash

rm -rf test-log/log-single
mkdir test-log/log-single
./bin/doppelganger -concurrency 6 -dsn1 "root:@tcp(172.17.0.1:4000)/sqlsmith" -log ./test-log/log-single -stable
