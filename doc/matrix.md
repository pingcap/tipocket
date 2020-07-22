# Matrix Support

[Matrix](https://github.com/chaos-mesh/matrix) is a config generator helpful for fuzzing.

TiPocket has integrated with Matrix and supports to use Matrix-format config generation.

## Usage

Matrix generates a set of config files based on its own config.

TiPocket can use Matrix-generated config files to initial TiDB cluster and setup system variables if corresponding file has been specified.

Note if TiPocket was initialed with a running cluster (with `-tidb-server` and so on), only `-matrix-sql` will take effect.

The [example](/config/matrix.yaml) generates 5 files, including config of TiDB, TiKV, PD, and two SQL files of system variables.

## CLI Arguments

```
bin/${testcase} -matrix-config ${matrix-config} -matrix-tidb ${matrix-tidb} -matrix-tikv ${matrix-tikv} -matrix-pd ${matrix-pd} -matrix-sql ${matrix-sql}
```

With the example Matrix config, TiPocket could be used with following arguments:
```
bin/${testcase} ${args} -matrix-config config/matrix.yaml \
 -matrix-tidb tidb.toml -matrix-tikv tikv.toml -matrix-pd pd.toml \
 -matrix-sql mysql-system-vars.sql -matrix-sql tidb-system-vars.sql
```

If you would like to explicitly disable (some of) Matrix-generated config, set `-matrix-*` to empty string will disable corresponding features.