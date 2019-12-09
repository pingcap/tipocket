package util

import (
	"time"
	"bytes"
	"github.com/pingcap/parser/ast"
	"github.com/pingcap/parser/format"
	"github.com/ngaut/log"
)

// BufferOut parser ast node to SQL string
func BufferOut(node ast.Node) (string, error) {
	out := new(bytes.Buffer)
	err := node.Restore(format.NewRestoreCtx(format.RestoreStringDoubleQuotes, out))
	if err != nil {
		return "", err
	}
	return string(out.Bytes()), nil
}

// TimeMustParse wrap time.Parse and panic when error
func TimeMustParse(layout, value string) time.Time {
	t, err := time.Parse(layout, value)
	if err != nil {
		log.Fatalf("parse time err %+v, layout: %s, value: %s", err, layout, value)
	}
	return t
}
