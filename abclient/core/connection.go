package core

import (
	"github.com/juju/errors"
)

func (e *Executor) reConnect() error {
	e.Lock()
	defer e.Unlock()
	for _, executor := range e.executors {
		if err := executor.ReConnect(); err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}
