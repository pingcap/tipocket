package control

import (
	"fmt"
	"os"
	"path"
	"sync"
	"time"

	"github.com/ngaut/log"

	"github.com/pingcap/tipocket/pkg/cluster"
)

var (
	minTime = time.Unix(0, 0)
)

// PanicCheck impls Plugin
type PanicCheck struct {
	*Controller

	// maps from string to *os.File
	podLogFiles   sync.Map
	lastQueryTime time.Time
}

// InitPlugin ...
func (c *PanicCheck) InitPlugin(control *Controller) {
	c.Controller = control
	c.lastQueryTime = minTime

	if c.lokiClient == nil {
		log.Warn("plugin panic check won't work")
		return
	}

	log.Info("panic check is running...")
	longDur := time.Minute * 10
	shortDur := time.Minute * 1
	panicTicker := time.NewTicker(longDur)
	// log ticker is a ticker for collecting pod logs, runs every 4 minute
	logTicker := time.NewTicker(shortDur)

	go func() {
		for {
			select {
			case <-c.ctx.Done():
				c.fetchTiDBClusterLogs(c.cfg.Nodes, time.Now())
				// Close all pod logs files
				c.podLogFiles.Range(func(key, value interface{}) bool {
					f := value.(*os.File)
					f.Close()
					return true
				})
				logTicker.Stop()
				panicTicker.Stop()
				log.Info("panic check is finished")
				return
			case <-panicTicker.C:
				c.checkTiDBClusterPanic(longDur, c.cfg.Nodes)
			case <-logTicker.C:
				// Here we don't use time.Now() is to prevent
				// the case that agent doesn't upload the latest logs
				c.fetchTiDBClusterLogs(c.cfg.Nodes, time.Now().Add(-2*time.Second))
			}
		}
	}()
}

func (c *PanicCheck) checkTiDBClusterPanic(dur time.Duration, nodes []cluster.Node) {
	wg := &sync.WaitGroup{}
	to := time.Now()
	from := to.Add(-dur)

	for _, n := range nodes {
		switch n.Component {
		case cluster.TiDB, cluster.TiKV, cluster.PD:
			wg.Add(1)
			var nonMatch []string
			if n.Component == cluster.TiKV {
				nonMatch = []string{
					"panic-when-unexpected-key-or-data",
					"No space left on device",
				}
			}

			// containerName is added to be sure that we only fetch the logs of
			// the container we want, not included sidecar containers.
			go func(ns, podName, containerName string, nonMatch []string) {
				defer wg.Done()
				texts, err := c.lokiClient.FetchPodLogs(ns, podName, containerName,
					"panic", nonMatch, from, to, 1000, false)
				if err != nil {
					log.Infof("failed to fetch logs from loki for pod %s in ns %s", podName, ns)
				} else if len(texts) > 0 {
					panicLogs := "panic-logs"
					file, err := os.OpenFile(path.Join(c.logPath, panicLogs),
						os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
					if err != nil {
						log.Fatal("failed to create panic logs file", err)
					}
					f, ok := c.podLogFiles.LoadOrStore(panicLogs, file)
					if ok {
						file.Close()
					}
					file = f.(*os.File)
					content := fmt.Sprintf("%d panics occurred in ns: %s pod %s. Content: %v", len(texts), ns, podName, texts)
					if _, err := file.WriteString(content); err != nil {
						log.Fatal("fail to write logs to panic logs file", content, err)
					}
				}
			}(n.Namespace, n.PodName, string(n.Component), nonMatch)
		default:
		}
	}
	wg.Wait()
}

func (c *PanicCheck) fetchTiDBClusterLogs(nodes []cluster.Node, toTime time.Time) {
	wg := &sync.WaitGroup{}

	for _, n := range nodes {
		switch n.Component {
		case cluster.TiDB, cluster.TiKV, cluster.PD:
			wg.Add(1)

			// containerName is added to be sure that we only fetch the logs of
			// the container we want, not included sidecar containers.
			go func(ns, podName, containerName string) {
				defer wg.Done()
				texts, err := c.lokiClient.FetchPodLogs(ns, podName, containerName,
					"", nil, c.lastQueryTime, toTime, 20000, false)
				if err != nil {
					log.Infof("failed to fetch logs from loki for pod %s in ns %s error is %v", podName, ns, err)
				} else {
					log.Infof("collect %s pod logs successfully, %d lines total", podName, len(texts))
				}
				file, err := os.OpenFile(path.Join(c.logPath, podName),
					os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if err != nil {
					log.Fatalf("failed to create log file for pod %s error is %v", podName, err)
				}
				f, ok := c.podLogFiles.LoadOrStore(podName, file)
				if ok {
					file.Close()
				}
				file = f.(*os.File)
				for _, line := range texts {
					if _, err = file.Write([]byte(line)); err != nil {
						log.Fatal("fail to write log file for pod %s error is %v", podName, err)
					}
				}
			}(n.Namespace, n.PodName, string(n.Component))
		default:
		}
	}
	wg.Wait()
	c.lastQueryTime = toTime
}
