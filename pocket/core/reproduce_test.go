package core

import (
	"testing"
	"github.com/pingcap/tipocket/pocket/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestParseExecNumber(t *testing.T) {
	assert.Equal(t, parseExecNumber("./test-log/log/ab-test.log"), 0)
	assert.Equal(t, parseExecNumber("./test-log/log/ab-test-1.log"), 1)
	assert.Equal(t, parseExecNumber("./test-log/log/ab-test-123.log"), 123)
}

func TestParseLog(t *testing.T) {
	var (
		log *types.Log
		err error
	)

	log, err = parseLog(`[2019/11/27 10:51:34.416 +08:00] [TODO] Exec SQL BEGIN`, 1)
	assert.Equal(t, err, nil)
	assert.Equal(t, log.GetNode(), 1)
	assert.Equal(t, log.GetSQL().SQLType, types.SQLTypeTxnBegin)
	assert.Equal(t, log.GetSQL().SQLStmt, "BEGIN")
	assert.Equal(t, log.GetTime().UnixNano(), int64(1574823094416000000))

	log, err = parseLog(`[2019/11/27 10:51:22.479 +08:00] [TODO] Exec SQL UPDATE sqlsmith.rpsfxc SET rpsfxc.vpmnuqnu=1614015085, rpsfxc.alvfuz=588064950, rpsfxc.soxjg=139378397, rpsfxc.bslhqvyn=421707098 WHERE rpsfxc.qhsro='2013-04-18 15:35:50' XOR rpsfxc.soxjg>351245791 OR rpsfxc.qhsro>'2053-07-09 00:00:31' XOR rpsfxc.obrdzbyj<_binary"4gJBz@]"`, 2)
	assert.Equal(t, err, nil)
	assert.Equal(t, log.GetNode(), 2)
	assert.Equal(t, log.GetSQL().SQLType, types.SQLTypeDMLUpdate)
	assert.Equal(t, log.GetSQL().SQLStmt, `UPDATE sqlsmith.rpsfxc SET rpsfxc.vpmnuqnu=1614015085, rpsfxc.alvfuz=588064950, rpsfxc.soxjg=139378397, rpsfxc.bslhqvyn=421707098 WHERE rpsfxc.qhsro='2013-04-18 15:35:50' XOR rpsfxc.soxjg>351245791 OR rpsfxc.qhsro>'2053-07-09 00:00:31' XOR rpsfxc.obrdzbyj<_binary"4gJBz@]"`)
	assert.Equal(t, log.GetTime().UnixNano(), int64(1574823082479000000))
}
