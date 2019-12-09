# Doppelgänger

An SQL abtest/test framework.

## Feature

* Transaction support for multi connections

* Replay log

## Build

```
make
```

## Usage

```
./bin/doppelganger -h
Usage of ./bin/doppelganger:
  -V	print version
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
    	reproduce from log, path:line, will execute to the line number, will not execute the given line
  -schema
    	print schema and exit
  -stable
    	generate stable SQL without random or other env related expression
```

- ABtest Example

```
./bin/doppelganger -dsn1 "root:@tcp(127.0.0.1:3306)/sqlsmith" -dsn2 "root@tcp(127.0.0.1:4000)/sqlsmith" -log ./log -stable -concurrency 3
```

With empty log parameter, all logs will be print to terminal.
