package types

import (
	"time"

	"github.com/juju/errors"
)

// Duration wrapper
type Duration struct {
	time.Duration
}

// UnmarshalText implement time.ParseDuration function for Duration
func (d *Duration) UnmarshalText(text []byte) error {
	var err error
	d.Duration, err = time.ParseDuration(string(text))
	return errors.Trace(err)
}
