package connection

import (
	"time"
	"github.com/ngaut/log"
)

func (c *Connection) logSQL(sql string, duration time.Duration, err error) {
	if err := c.logger.Infof("Success: %t, Duration: %s, SQL: %s", err == nil, duration, sql); err != nil {
		log.Fatalf("fail to log to file %v", err)
	}
}
