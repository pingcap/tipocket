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

const (
	logLimit = 10000
)

var (
	minTime = time.Unix(0, 0)
)

// PanicCheck impls Plugin
type PanicCheck struct {
	*Controller
	PanicIf bool
	// maps from string to *os.File
	lastQueryTime time.Time
}

// PanicRecord records panic logs
type PanicRecord struct {
	Node *cluster.Node
	Logs []string
}

// PanicRecords is []PanicRecord
type PanicRecords []PanicRecord

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
	c.Controller = control
	c.lastQueryTime = minTime

	if c.logsClient == nil {
		log.Warn("plugin panic check won't work since logs client is nil")
		return
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
					if c.PanicIf {
						log.Fatalf("%s\n", records.String())
					} else {
						log.Warnf("%s\n", records.String())
					}
				}
			}
		}
	}()
}

func (c *PanicCheck) checkTiDBClusterPanic(ctx context.Context, nodes []cluster.Node) PanicRecords {
	var err error
	to, from := time.Now(), c.lastQueryTime
	wg, mu, panicRecords := sync.WaitGroup{}, sync.Mutex{}, PanicRecords(nil)

	defer func() {
		if err == nil {
			c.lastQueryTime = to
		}
	}()

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
			wg.Add(1)
			go func() {
				defer wg.Done()
				logLines, err := c.Controller.logsClient.SearchLog(ctx, node.IP, int(node.Port),
					from, to,
					[]logs.LogLevel{logs.LogLevelCritical, logs.LogLevelDebug, logs.LogLevelError, logs.LogLevelInfo, logs.LogLevelTrace, logs.LogLevelUnknown, logs.LogLevelWarn},
					[]string{"Welcome"},
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
