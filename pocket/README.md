# Pocket

An SQL ab/sync test framework.

## Feature

* Transaction support for multi connections

* Replay from logs

## Build

```
make
```

## Usage

```
./bin/pocket -h
Usage of ./bin/pocket:
  -V    print version
  -binlog
        enable binlog test which requires 2 dsn points but only write first
  -clear
        drop all tables in target database and then start testing
  -concurrency int
        test concurrency (default 1)
  -dsn1 string
        dsn1
  -dsn2 string
        dsn2
  -log string
        log path
  -re string
        reproduce from logs, dir:table, will execute only the given table unless all table
  -schema
        print schema and exit
  -stable
        generate stable SQL without random or other env related expression
```

- ABtest Example

```
./bin/pocket -concurrency 3 -dsn1 "root:@tcp(127.0.0.1:3306)/sqlsmith" -dsn2 "root@tcp(127.0.0.1:4000)/sqlsmith" -log ./log -stable
```

With empty log parameter, all logs will be print to terminal.
