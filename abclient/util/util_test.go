package util

import (
	"testing"
	"time"
	"github.com/stretchr/testify/assert"
)

func TestFormatTimeStrAsLog(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	assert.Equal(t, "2019/08/10 11:45:14.000 +08:00", formatTimeStrAsLog(time.Unix(1565408714, 0).In(loc)))
}
