package util

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/chaos-mesh/matrix/api"
	"github.com/ngaut/log"

	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	"github.com/pingcap/tipocket/pkg/test-infra/tidb"
	"github.com/pingcap/tipocket/util"
)

func matrixnize(c *cluster.Specs) (bool, func([]cluster.Node) error, func(), error) {
	var tiDBConfig *fixture.TiDBClusterConfig
	if c, ok := c.Cluster.(*tidb.Ops); ok {
		tiDBConfig = c.GetTiDBConfig()
	} else {
		log.Info("Not TiDB cluster, skip Matrix.")
		return false, nil, nil, nil
	}

	matrixedConfig := *tiDBConfig

	var setupNodes func([]cluster.Node) error
	var cleaner func()

	var matrixSQLStmts []string
	if tiDBConfig.MatrixConfig.MatrixConfigFile == "" {
		return false, nil, nil, nil
	}

	matrixCtx, err := ioutil.TempDir("", "matrix")
	if err != nil {
		log.Warn(fmt.Sprintf("Failed to create Matrix context folder: `%s`, skip Matrix.", err.Error()))
		return false, nil, nil, err
	}
	cleaner = func() {
		if !tiDBConfig.MatrixConfig.NoCleanup {
			_ = os.RemoveAll(matrixCtx)
		}
	}

	err = api.Gen(tiDBConfig.MatrixConfig.MatrixConfigFile, matrixCtx, 0)
	if err != nil {
		log.Error("Matrix generation failed.")
		return false, nil, nil, err
	}
	checkConfigEnabledAndOverwrite := func(matrixConfig string, realConfig *string) (bool, error) {
		if matrixConfig != "" {
			joinedMatrixConfig := path.Join(matrixCtx, matrixConfig)
			if util.IsFileExist(joinedMatrixConfig) {
				if *realConfig != "" {
					return false, errors.New(fmt.Sprintf("Target config file already specified: `%s`", *realConfig))
				}
				*realConfig = joinedMatrixConfig
				return true, nil
			}
			return false, errors.New(fmt.Sprintf("`%s` not exists in Matrix output", matrixConfig))
		}
		return false, nil
	}

	var matrixTiDB, matrixTiKV, matrixPD, matrixSQL bool
	if matrixTiDB, err = checkConfigEnabledAndOverwrite(tiDBConfig.MatrixConfig.MatrixTiDBConfig, &matrixedConfig.TiDBConfig); err != nil {
		return false, nil, nil, err
	}
	if matrixTiKV, err = checkConfigEnabledAndOverwrite(tiDBConfig.MatrixConfig.MatrixTiKVConfig, &matrixedConfig.TiKVConfig); err != nil {
		return false, nil, nil, err
	}
	if matrixPD, err = checkConfigEnabledAndOverwrite(tiDBConfig.MatrixConfig.MatrixPDConfig, &matrixedConfig.PDConfig); err != nil {
		return false, nil, nil, err
	}

	if len(tiDBConfig.MatrixConfig.MatrixSQLConfig) > 0 {
		matrixSQL = true

		for _, sqlFile := range tiDBConfig.MatrixConfig.MatrixSQLConfig {
			sqlFile = path.Join(matrixCtx, sqlFile)
			b, err := ioutil.ReadFile(sqlFile)
			if err != nil {
				log.Warn(fmt.Sprintf("Error loading from Matrix: %s", err.Error()))
				matrixSQLStmts = nil
				matrixSQL = false
				break
			}
			matrixSQLStmts = append(matrixSQLStmts, string(b))
		}
	}
	if !(matrixTiDB || matrixTiKV || matrixPD || matrixSQL) {
		return false, nil, nil, errors.New("`Matrix` enabled but no output from Matrix is used")
	}
	if matrixTiDB {
		log.Info("Use TiDB config from Matrix")
	}
	if matrixTiKV {
		log.Info("Use TiKV config from Matrix")
	}
	if matrixPD {
		log.Info("Use PD config from Matrix")
	}
	if matrixSQL {
		log.Info("Use SQL statements from Matrix")
	}

	setupNodes = func(nodes []cluster.Node) error {
		for _, node := range nodes {
			if node.Component == cluster.TiDB {
				dsn := fmt.Sprintf("root@tcp(%s:%d)/", node.IP, node.Port)
				db, err := util.OpenDB(dsn, 1)
				if err != nil {
					return err
				}
				//noinspection GoDeferInLoop
				defer db.Close()
				for _, stmt := range matrixSQLStmts {
					_, err = db.Exec(stmt)
					if err != nil {
						return err
					}
				}
			}
		}
		return nil
	}

	// no error, overwrite original config
	tiDBConfig.TiDBConfig = matrixedConfig.TiDBConfig
	tiDBConfig.TiKVConfig = matrixedConfig.TiKVConfig
	tiDBConfig.PDConfig = matrixedConfig.PDConfig
	return true, setupNodes, cleaner, nil
}
