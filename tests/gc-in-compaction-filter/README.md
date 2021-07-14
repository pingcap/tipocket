### Prepare Data

```
$ sysbench --mysql-host=<host> --mysql-port=<port> --mysql-user=root --tables=128 --table-size=1000000 --threads=32 --time=600 common.lua prepare
```

### Run the Test Case

```
./run.sh <tidb-host> <tidb-port>
```

### TODO list
- [ ] support to automatic run it on tipocket
