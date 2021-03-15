package control

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ngaut/log"

	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/logs"
)

var (
	logLimit        = 10000
	minTime         = time.Unix(0, 0)
	_        Plugin = &PanicCheck{}
)

// PanicCheck impls Plugin
type PanicCheck struct {
	*Controller
	silent bool
	// maps from string to *os.File
	lastQueryTimeM sync.Map
}

// NewPanicCheck creates an panic check instance
func NewPanicCheck(silent bool) *PanicCheck {
	return &PanicCheck{silent: silent}
}

// PanicRecord records panic logs
type PanicRecord struct {
	Node *cluster.Node
	Logs []string
}

// PanicRecords is []PanicRecord
type PanicRecords []PanicRecord

// String ...
func (e PanicRecords) String() string {
	var buf strings.Builder
	buf.WriteString(fmt.Sprint("panic found:\n"))
	for _, record := range e {
		for i, log := range record.Logs {
			if i == 0 {
				buf.WriteString(fmt.Sprintf("\t%s:\n", record.Node.String()))
			}
			buf.WriteString(fmt.Sprintf("\t\t%s\n", log))
		}
	}
	return buf.String()
}

// InitPlugin ...
func (c *PanicCheck) InitPlugin(control *Controller) {
	if control.logsClient == nil {
		log.Warn("plugin panic check won't work since logs client is nil")
		return
	}
	c.Controller = control
	for _, node := range control.cfg.Nodes {
		c.lastQueryTimeM.Store(node.String(), minTime)
	}

	log.Info("panic checker is running...")
	panicTicker := time.NewTicker(time.Minute)
	go func() {
		for {
			select {
			case <-c.ctx.Done():
				panicTicker.Stop()
				log.Info("panic checker is finished")
				return
			case <-panicTicker.C:
				records := c.checkTiDBClusterPanic(c.Controller.ctx, c.cfg.Nodes)
				if records != nil {
					if c.silent {
						log.Warnf("%s\n", records.String())
					} else {
						log.Fatalf("%s\n", records.String())
					}
				}
			}
		}
	}()
}

func (c *PanicCheck) checkTiDBClusterPanic(ctx context.Context, nodes []cluster.Node) PanicRecords {
	wg, mu, panicRecords := sync.WaitGroup{}, sync.Mutex{}, PanicRecords(nil)

	now := time.Now()
	for _, n := range nodes {
		switch n.Component {
		case cluster.TiDB, cluster.TiKV, cluster.PD:
			var nonMatch []string
			if n.Component == cluster.TiKV {
				nonMatch = []string{
					"panic-when-unexpected-key-or-data",
					"No space left on device",
				}
			}
			node := n
			panicRecord := PanicRecord{Node: &node}

			from, _ := c.lastQueryTimeM.Load(node.String())
			wg.Add(1)
			go func() {
				defer func() {
					c.lastQueryTimeM.Store(node.String(), now)
					wg.Done()
				}()
				ip, port := node.IP, int(node.Port)
				if node.Component == cluster.TiDB {
					port = 10080
				}
				logLines, err := c.Controller.logsClient.SearchLog(ctx,
					ip, port,
					from.(time.Time), now,
					[]logs.LogLevel{logs.LogLevelCritical, logs.LogLevelDebug, logs.LogLevelError, logs.LogLevelInfo, logs.LogLevelTrace, logs.LogLevelUnknown, logs.LogLevelWarn},
					[]string{"(?i)panic"},
					logLimit,
				)
				if err != nil {
					log.Warningf("search logs failed on %s: %v", node.String(), err)
					return
				}
			ScanLog:
				for _, log := range logLines {
					for _, ignore := range nonMatch {
						if strings.Contains(log, ignore) {
							continue ScanLog
						}
					}
					panicRecord.Logs = append(panicRecord.Logs, log)
				}
				if len(panicRecord.Logs) != 0 {
					mu.Lock()
					panicRecords = append(panicRecords, panicRecord)
					mu.Unlock()
				}
			}()
		}
	}
	wg.Wait()
	return panicRecords
}
