package main

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/pingcap/test-infra/sdk/core"
	"github.com/pingcap/test-infra/sdk/resource"
	"go.uber.org/zap"

	apiresource "github.com/pingcap/test-infra/common/model/resource"

	. "github.com/tipocket/testcase/sdk-ticase-4885"
)

// This testcase automates the TICASE-4885
//
// Required resources, their usage should all conforms to its own README:
//  - tc
//  - br
//  - go_tpc
//
// There are several parameters should be injected from outside:
//
//  - tpcc_br_storage_uri: Location of BR backup. Defaults to s3://tpcc/br-2t.
//  - tpcc_br_s3_endpoint: --s3.endpoint of BR. Defaults to http://minio.pingcap.net:9000.
//  - tpcc_warehouses: --warehouses of go-tpc. Defaults to 4.
//  - tpcc_tables: -T of go-tpc. Defaults to 8.
//  - scale_instance_type: During the two scaling phase, which component should be scaled. e.g. tikv, tidb. Required.
//  - scale_size: During the two scaling phase, how many instances should be scaled. Required.
//  - scale_timeout: Timeout of the two scaling phase. Defaults to 5m.
//  - upgrade_target: During the upgrade phase, which component should be updated. e.g. tikv, tidb. Required.
//  - upgrade_image: The component will be updated to a specified image. e.g. pingcap/tidb:5.0.1. Required.
//  - phase_interval: Sleeping interval between three phases. Defaults to 3h.
//  - metric_interval: Metrics listener will flush metrics and reevaluate constraints every metric_interval time. Defaults to 1m.
//
func main() {
	Try(godotenv.Load())
	ctx := Try(core.BuildContext()).(core.TestContext)
	log := ctx.Logger()
	log.Info("start initializing environment...")

	// Initialize param
	brUri := ctx.Param("tpcc_br_storage_uri", "s3://tpcc/br-2t")
	brEndpoint := ctx.Param("tpcc_br_s3_endpoint", "http://minio.pingcap.net:9000")
	tpccWarehouses := Try(strconv.Atoi(ctx.Param("tpcc_warehouses", "4"))).(int)
	tpccTables := Try(strconv.Atoi(ctx.Param("tpcc_tables", "8"))).(int)

	phaseInterval := Try(time.ParseDuration(ctx.Param("phase_interval", "3h"))).(time.Duration)
	metricInterval := Try(time.ParseDuration(ctx.Param("metric_interval", "1m"))).(time.Duration)

	scaleInstanceType := InstanceType(ctx.Param("scale_instance_type"))
	scaleSize, _ := Try(strconv.ParseUint(ctx.Param("scale_size"), 10, 32)).(uint)
	scaleTimeout := Try(time.ParseDuration(ctx.Param("scale_timeout", "5m"))).(time.Duration)
	upgradeTarget := InstanceType(ctx.Param("upgrade_target"))
	upgradeImage := ctx.Param("upgrade_image")

	// Initialize resource
	target := ctx.Resource("tc").(resource.TiDBCluster)
	br := ctx.Resource("br").(resource.BR)
	tpc := ctx.Resource("go_tpc").(resource.WorkloadNode)

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
	err = Exec("go_tpc", tpc, resource.WorkloadNodeExecOptions{
		Command: fmt.Sprintf("/go-tpc tpcc --warehouses %d run -T %d", tpccWarehouses, tpccTables),
	}, log)
	if err != nil {
		return
	}

	stopper := make(chan int)
	go func() {
		// scale out
		replicas := getReplicas(target, scaleInstanceType)
		scale := resource.TiDBClusterScaleOptions{
			Target:   scaleInstanceType,
			Replicas: replicas + uint32(scaleSize),
			Timeout:  scaleTimeout,
		}
		log.Info("now scaling out.", zap.Any("scale_options", scale))
		err := target.Scale(scale)
		if err != nil {
			log.Error("errored when scaling out", zap.Any("scale_options", scale), zap.Error(err))
		}
		time.Sleep(phaseInterval)
		log.Info("phase scaling out executed. now sleeping...", zap.Duration("phase_interval", phaseInterval))

		// scale in
		replicas = getReplicas(target, scaleInstanceType)
		scale = resource.TiDBClusterScaleOptions{
			Target:   scaleInstanceType,
			Replicas: replicas - uint32(scaleSize),
			Timeout:  scaleTimeout,
		}
		log.Info("now scaling in.", zap.Any("scale_options", scale))
		err = target.Scale(scale)
		if err != nil {
			log.Error("errored when scaling in", zap.Any("scale_options", scale), zap.Error(err))
		}
		time.Sleep(phaseInterval)
		log.Info("phase in out executed. now sleeping...", zap.Duration("phase_interval", phaseInterval))

		upgrade := resource.TiDBClusterUpgradeOptions{
			Target: upgradeTarget,
			Image:  upgradeImage,
		}
		err = target.Upgrade(upgrade)
		if err != nil {
			log.Error("errored when upgrade", zap.Any("upgrade_options", upgrade), zap.Error(err))
		}
		time.Sleep(phaseInterval)
		log.Info("phase upgrade executed. now exiting...")

		stopper <- 0
	}()

	go ListenMetrics(Try(target.ServiceURL(resource.Prometheus)).(url.URL), metricInterval, log, stopper)
}

func getReplicas(target resource.TiDBCluster, scaleInstanceType resource.TiDBCompType) uint32 {
	return uint32(Try(target.GetSpec(scaleInstanceType)).(*apiresource.TiDBClusterComponentSpec).Replicas)
}
