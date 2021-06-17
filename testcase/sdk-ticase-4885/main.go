package main

import (
	"context"
	"errors"
	"fmt"
	. "net/url"
	"regexp"
	"strconv"
	"time"

	"github.com/pingcap/test-infra/sdk/core"
	"github.com/pingcap/test-infra/sdk/resource"
	prometheus "github.com/prometheus/client_golang/api"
	prometheus_api "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"go.uber.org/zap"
)

func main() {
	ctx := try(core.BuildContext()).(core.TestContext)
	log := ctx.Logger()
	log.Info("start initializing environment...")

	// Initialize param
	brUri := ctx.Param("tpcc-br-uri", "s3://tpcc/br-2t")
	tpccWarehouses, _ := strconv.Atoi(ctx.Param("tpcc-warehouses", "4"))
	tpccTables, _ := strconv.Atoi(ctx.Param("tpcc-tables", "4"))
	instanceType := ctx.Param("instance-type")
	scaleSize, _ := strconv.ParseUint(ctx.Param("scale-size"), 10, 32)
	scaleTimeout, _ := time.ParseDuration(ctx.Param("scale-timeout", "5m"))

	// Initialize resource
	target := ctx.Resource("TiDBCluster").(resource.TiDBCluster)
	br := ctx.Resource("br").(resource.BR)
	tpc := ctx.Resource("go-tpc").(resource.WorkloadNode)

	log.Info("environment initialized. running BR to import data.")
	err := exec("br", br, resource.WorkloadNodeExecOptions{
		Command: fmt.Sprintf("./br restore full --pd \"%s\" --storage \"%s\" --s3.endpoint http://minio.pingcap.net:9000 --send-credentials-to-tikv=true",
			regexp.MustCompile("^http(s)://").ReplaceAllString(try(target.ServiceURL(resource.PDAddr)).(string), ""),
			brUri),
	}, log)
	if err != nil {
		return
	}

	log.Info("data imported. now running TPC-C test.")
	err = exec("go-tpc", tpc, resource.WorkloadNodeExecOptions{
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

	go func() {
		u := try(target.ServiceURL(resource.Prometheus)).(URL)
		url := u.String()
		client, err := prometheus.NewClient(prometheus.Config{
			Address: url,
		})
		if err != nil {
			log.Error("errored when create Prometheus client", zap.String("url", url), zap.Error(err))
			return
		}
		api := prometheus_api.NewAPI(client)

		tidbClusters, err := queryParams("sum(pd_cluster_status) by (tidb_cluster)", "tidb_cluster", api, log)
		if err != nil {
			return
		}

		// instances, err := queryParams("sum(pd_cluster_status) by (instance)", "instance", api, log)
		// if err != nil {
		// 	return
		// }

		start := time.Now()
		INTERVAL := 5 * time.Second // flush and reevaluate constraints every INTERVAL time.

		// QPS: sum(rate(tidb_executor_statement_total{tidb_cluster="testbed-tidbcluster-getstatus-jijbtjve-tc-ymcwmxhu"}[1m]))
		// Duration: histogram_quantile(0.80, sum(rate(tidb_server_handle_query_duration_seconds_bucket{tidb_cluster="testbed-tidbcluster-getstatus-jijbtjve-tc-ymcwmxhu"}[1m])) by (le))
		// Transaction OPS: sum(rate(tidb_session_transaction_duration_seconds_count{tidb_cluster="testbed-tidbcluster-getstatus-jijbtjve-tc-ymcwmxhu"}[1m])) by (type)
		// Up Store Count: sum(pd_cluster_status{tidb_cluster="testbed-tidbcluster-getstatus-jijbtjve-tc-ymcwmxhu", instance="tc-ymcwmxhu-pd-0", type="store_up_count"})

		// 将 “TPS” 理解为 “QPS”
		qpsQuery := fmt.Sprintf("sum(rate(tidb_executor_statement_total{tidb_cluster=\"%s\"}[1m]))", *tidbClusters)
		// 将 “lat” 理解为 “Duration 的 80% 位百分数”
		durationQuery := fmt.Sprintf("histogram_quantile(0.80, sum(rate(tidb_server_handle_query_duration_seconds_bucket{tidb_cluster=\"$tidb_cluster\"}[1m])) by (le))")

		for {
			select {
			case <-stopper:
				return
			default:
				log.Info("constraints reevaluating...")

				// QPS
				log.Info("retrieving Query Summary -- QPS...")
				qpssValue, err := queryRange(qpsQuery,
					prometheus_api.Range{
						Start: start,
						End:   time.Now(),
						Step:  INTERVAL,
					}, api, log)
				if err != nil {
					return
				}
				qpssMatrix, err := expectMatrix(qpsQuery, qpssValue, log)
				if err != nil {
					return
				}

				secondLast, latest := retrieveLastTwo("QPS", (*qpssMatrix[0]).Values, log)
				// 将 “TPS 不能下降到 三分之一 以下” 理解为 “最新一条记录 不能下降到 倒数第二条记录 的三分之一及以下”
				// 即 “倒数第二条记录 是 最新一条记录 的三倍及以上时，限制不满足”
				if secondLast >= 3.0*latest {
					log.Error("[CONSTRAINT VIOLATED] QOS constraint breached.")
				}

				// Duration
				log.Info("retrieving Query Summary -- 80% Duration")
				durationValue, err := queryRange(durationQuery,
					prometheus_api.Range{
						Start: start,
						End:   time.Now(),
						Step:  INTERVAL,
					}, api, log)
				if err != nil {
					return
				}
				durationMatrix, err := expectMatrix(durationQuery, durationValue, log)
				if err != nil {
					return
				}

				secondLast, latest = retrieveLastTwo("80% Duration", (*durationMatrix[0]).Values, log)
				// 将 “lat 不能升高 10% 以上” 理解为 “最新一条记录 不能升高到 倒数第二条记录 的百分之十及以上”
				// 即 “最新一条记录 是 倒数第二条记录 的 1.1 倍及以上时，限制不满足”
				if latest >= secondLast*1.1 {
					log.Error("[CONSTRAINT VIOLATED] 80% Duration constraint breached.")
				}
			}
		}
	}()
}

func retrieveLastTwo(metric string, samplePairs []model.SamplePair, log *zap.Logger) (model.SampleValue, model.SampleValue) {
	secondLast := samplePairs[len(samplePairs)-2]
	latest := samplePairs[len(samplePairs)-1]
	log.Info(metric+" retrieved.", zap.Any("second_last_value", secondLast), zap.Any("latest_value", latest))
	return secondLast.Value, latest.Value
}

func queryParams(q string, labelName model.LabelName, api prometheus_api.API, log *zap.Logger) (*model.LabelValue, error) {
	value, err := queryNow(q, api, log)
	if err != nil {
		return nil, err
	}
	vector, err := expectVector(q, value, log)
	if err != nil {
		return nil, err
	}

	if len(vector) != 1 {
		msg := "a single element vector expected for given query, yet a vector whose length is not 1 returned"
		log.Error(msg,
			zap.String("query", q),
			zap.Any("vector", vector))
		return nil, errors.New(msg)
	} else {
		labelValue := vector[0].Metric[labelName]
		return &labelValue, nil
	}
}

func expectVector(q string, value model.Value, log *zap.Logger) (model.Vector, error) {
	if value.Type() != model.ValVector {
		msg := "a vector expected for given query, but a non-Vector returned"
		log.Error(msg,
			zap.String("query", q),
			zap.Any("type", value.Type()),
			zap.String("value", value.String()))
		return nil, errors.New(msg)
	}
	return value.(model.Vector), nil
}

func expectMatrix(q string, value model.Value, log *zap.Logger) (model.Matrix, error) {
	if value.Type() != model.ValMatrix {
		msg := "a matrix expected for given query, but a non-Vector returned"
		log.Error(msg,
			zap.String("query", q),
			zap.Any("type", value.Type()),
			zap.String("value", value.String()))
		return nil, errors.New(msg)
	}
	return value.(model.Matrix), nil
}

func queryRange(query string, rg prometheus_api.Range, api prometheus_api.API, log *zap.Logger) (model.Value, error) {
	value, warnings, err := api.QueryRange(context.Background(), query, rg)
	if err != nil {
		log.Error("errored when executing query against Prometheus", zap.String("query", query), zap.Error(err))
		return nil, err
	}
	if len(warnings) > 0 {
		log.Warn("warning returned when executing query against Prometheus", zap.String("query", query), zap.Strings("warnings", warnings))
	}
	return value, nil
}

func queryNow(query string, api prometheus_api.API, log *zap.Logger) (model.Value, error) {
	value, warnings, err := api.Query(context.Background(), query, time.Now())
	if err != nil {
		log.Error("errored when executing query against Prometheus", zap.String("query", query), zap.Error(err))
		return nil, err
	}
	if len(warnings) > 0 {
		log.Warn("warning returned when executing query against Prometheus", zap.String("query", query), zap.Strings("warnings", warnings))
	}
	return value, nil
}

func exec(description string, workload resource.WorkloadNode, options resource.WorkloadNodeExecOptions, log *zap.Logger) error {
	_, stderr, exitCode, err := workload.Exec(options)
	if err != nil {
		log.Error("errored when executing command workload", zap.String("description", description), zap.Any("options", options), zap.Error(err))
		return err
	}
	if exitCode != 0 {
		// TODO: do log.Error writes into stderr?
		log.Error("errored returned from executed command",
			zap.String("description", description),
			zap.Any("options", options),
			zap.Int("exitCode", exitCode),
			zap.String("stderr", stderr))
		return errors.New("errored returned from executed command. description: " + description)
	}
	return nil
}

func InstanceType(str string) resource.TiDBCompType {
	comp, err := resource.TiDB.FromString(str)
	if err != nil {
		core.Fail("unsupported parameter instance-type " + str)
		return -1
	}
	return comp
}

func try(xs ...interface{}) interface{} {
	if len(xs) == 0 {
		return nil
	}

	if err, ok := xs[len(xs)-1].(error); ok && err != nil {
		core.Fail(err.Error())
	}

	return xs[0]
}
