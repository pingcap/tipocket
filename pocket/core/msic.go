package core

import (
	"github.com/juju/errors"
)

// PrintSchema print schema information and return
func (e *Executor) PrintSchema() error {
	return errors.Trace(e.executors[0].PrintSchema())
}

