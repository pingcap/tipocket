package util

import (
	"strings"

	"github.com/pingcap/errors"
)

func WrapErrors(errs []error) error {
	if len(errs) < 1 {
		return nil
	}
	var msgs []string
	for _, err := range errs {
		msgs = append(msgs, err.Error())
	}
	return errors.New(strings.Join(msgs, ","))
}
