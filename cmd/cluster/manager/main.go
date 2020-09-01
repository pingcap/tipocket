package main

import (
	"fmt"

	"github.com/juju/errors"
	"github.com/spf13/cobra"

	"github.com/pingcap/tipocket/pkg/cluster/manager"
	"github.com/pingcap/tipocket/pkg/cluster/manager/util"
	"github.com/pingcap/tipocket/pkg/logger"
)

var (
	dbName   string
	host     string
	port     int
	user     string
	password string
)

func main() {
	logger.InitGlobalLogger()
	var rootCmd = &cobra.Command{
		Use:   "manager",
		Short: "Cluster manager",
		RunE: func(cmd *cobra.Command, args []string) error {
			dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true", user, password, host, port, dbName)
			mgr, err := manager.New(dsn)
			if err != nil {
				return errors.Trace(err)
			}
			if err := mgr.Run(); err != nil {
				return errors.Trace(err)
			}
			return nil
		},
	}
	rootCmd.PersistentFlags().StringVarP(&dbName, "db", "D", "tipocket", "Database name")
	rootCmd.PersistentFlags().StringVarP(&host, "host", "H", "127.0.0.1", "Database host")
	rootCmd.PersistentFlags().IntVarP(&port, "port", "P", 4000, "Database port")
	rootCmd.PersistentFlags().StringVarP(&user, "user", "u", "root", "Database user")
	rootCmd.PersistentFlags().StringVarP(&password, "password", "p", "", "Database password")
	rootCmd.PersistentFlags().StringVarP(&util.Addr, "addr", "A", "127.0.0.1:8000", "The address that the manager monitors")
	rootCmd.PersistentFlags().StringVar(&util.S3Endpoint, "s3.endpoint", "", "The endpoint of s3")
	rootCmd.PersistentFlags().StringVar(&util.AwsAccessKeyID, "s3.access-key", "YOURACCESSKEY", "The access key of s3")
	rootCmd.PersistentFlags().StringVar(&util.AwsSecretAccessKey, "s3.secret-key", "YOURSECRETKEY", "The secret key of s3")

	rootCmd.Execute()
}
