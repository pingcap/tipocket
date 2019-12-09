package util

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

const (
	layout = "2006-01-02 15:04:05"
)

func TestSQLSmith_TestIfDaylightTime(t *testing.T) {
	assert.Equal(t, ifDaylightTime(TimeMustParse(layout, "1986-05-05 11:45:14")), true)
	assert.Equal(t, ifDaylightTime(TimeMustParse(layout, "1991-09-05 11:45:14")), true)
	assert.Equal(t, ifDaylightTime(TimeMustParse(layout, "1985-08-05 11:45:14")), false)
	assert.Equal(t, ifDaylightTime(TimeMustParse(layout, "1992-06-05 11:45:14")), false)
}
