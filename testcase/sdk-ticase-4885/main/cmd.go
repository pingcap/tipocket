package main

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"time"

	"github.com/pingcap/test-infra/sdk/core"
	"github.com/pingcap/test-infra/sdk/resource"
	"go.uber.org/zap"

	. "github.com/tipocket/testcase/sdk-ticase-4885"
)

// This testcase automates the TICASE-4885
//
// Required resources, their usage should all conforms to its own README:
//  - TiDBCluster
//  - br
//  - go-tpc
//
// There are several parameters should be injected from outside:
//
//  - tpcc-br-storage-uri: Location of BR backup. Defaults to s3://tpcc/br-2t.
//  - tpcc-br-s3-endpoint: --s3.endpoint of BR. Defaults to http://minio.pingcap.net:9000.
//  - tpcc-warehouses: --warehouses of go-tpc. Defaults to 4.
//  - tpcc-tables: -T of go-tpc. Defaults to 8.
//  - scale-instance-type: During the two scaling phase, which component should be scaled. e.g. tikv, tidb. Required.
//  - scale-size: During the two scaling phase, how many instances should be scaled. Required.
//  - scale-timeout: Timeout of the two scaling phase. Defaults to 5m.
//  - upgrade-target: During the upgrade phase, which component should be updated. e.g. tikv, tidb. Required.
//  - upgrade-image: The component will be updated to a specified image. e.g. pingcap/tidb:5.0.1. Required.
//
// And one constant directly coded into this program:
//  - INTERVAL: this program will flush metrics and reevaluate constraints every INTERVAL time.
func main() {
	ctx := Try(core.BuildContext()).(core.TestContext)
	log := ctx.Logger()
	log.Info("start initializing environment...")

	// Initialize param
	brUri := ctx.Param("tpcc-br-storage-uri", "s3://tpcc/br-2t")
	brEndpoint := ctx.Param("tpcc-br-s3-endpoint", "http://minio.pingcap.net:9000")
	tpccWarehouses, _ := strconv.Atoi(ctx.Param("tpcc-warehouses", "4"))
	tpccTables, _ := strconv.Atoi(ctx.Param("tpcc-tables", "8"))
	instanceType := ctx.Param("scale-instance-type")
	scaleSize, _ := strconv.ParseUint(ctx.Param("scale-size"), 10, 32)
	scaleTimeout, _ := time.ParseDuration(ctx.Param("scale-timeout", "5m"))

	// Initialize resource
	target := ctx.Resource("TiDBCluster").(resource.TiDBCluster)
	br := ctx.Resource("br").(resource.BR)
	tpc := ctx.Resource("go-tpc").(resource.WorkloadNode)

	log.Info("environment initialized. running BR to import data.")
	err := Exec("br", br, resource.WorkloadNodeExecOptions{
		Command: fmt.Sprintf("./br restore full --pd \"%s\" --storage \"%s\" --s3.endpoint \"%s\" --send-credentials-to-tikv=true",
			regexp.MustCompile("^http(s)://").ReplaceAllString(Try(target.ServiceURL(resource.PDAddr)).(string), ""),
			brEndpoint,
			brUri),
	}, log)
	if err != nil {
		return
	}

	log.Info("data imported. now running TPC-C test.")
	err = Exec("go-tpc", tpc, resource.WorkloadNodeExecOptions{
		Command: fmt.Sprintf("./bin/go-tpc tpcc --warehouses %d run -T %d", tpccWarehouses, tpccTables),
	}, log)
	if err != nil {
		return
	}

	stopper := make(chan int)
	go func() {
		// scale out
		scale := resource.TiDBClusterScaleOptions{
			Target:   InstanceType(instanceType),
			Replicas: uint32(scaleSize), // TODO: get spec?
			Timeout:  scaleTimeout,
		}
		log.Info("now scaling out.", zap.Any("scale_options", scale))
		err := target.Scale(scale)
		if err != nil {
			log.Error("errored when scaling out", zap.Any("scale_options", scale), zap.Error(err))
		}
		time.Sleep(3 * time.Hour)

		// scale in
		scale = resource.TiDBClusterScaleOptions{
			Target:   InstanceType(instanceType),
			Replicas: uint32(scaleSize), // TODO: get spec?
			Timeout:  scaleTimeout,
		}
		log.Info("now scaling in.", zap.Any("scale_options", scale))
		err = target.Scale(scale)
		if err != nil {
			log.Error("errored when scaling in", zap.Any("scale_options", scale), zap.Error(err))
		}
		time.Sleep(3 * time.Hour)

		upgrade := resource.TiDBClusterUpgradeOptions{
			Target: InstanceType(ctx.Param("upgrade-target")),
			Image:  ctx.Param("upgrade-image"),
		}
		err = target.Upgrade(upgrade)
		if err != nil {
			log.Error("errored when upgrade", zap.Any("upgrade_options", upgrade), zap.Error(err))
		}
		time.Sleep(3 * time.Hour)

		stopper <- 0
	}()

	go ListenMetrics(Try(target.ServiceURL(resource.Prometheus)).(url.URL), 5*time.Second, log, stopper)
}
