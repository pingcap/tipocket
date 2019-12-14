package core

import (
	"math/rand"
	"time"
	"sync"
	"github.com/ngaut/log"
	"github.com/pingcap/tipocket/pocket/executor"
	"github.com/pingcap/tipocket/pocket/pkg/types"
)

const (
	maxExecuteTime = 4.0 // Second
)

func (e *Executor) watchDeadLock() {
	lastExecTime := time.Now()
	go func() {
		c := time.Tick(time.Second)
		for range c {
			if time.Now().Sub(lastExecTime).Seconds() > maxExecuteTime {
				// deadlock detected
				if e.ifLock {
					// if the core goroutine block in another lock, skip deadlock
					continue
				}
				e.Lock()
				log.Info("deadlock detected")
				e.resolveDeadLock()
				e.Unlock()
			}
		}
	}()
	for {
		e.order.Push(<- e.deadlockCh)
		lastExecTime = time.Now()
	}
}

func (e *Executor) resolveDeadLock() {
	// log.Info(e.order.GetHistroy())
	wg := sync.WaitGroup{}
	for e.order.Next() {
		for _, executor := range e.executors {
			if executor.GetID() == e.order.Val() {
				time.Sleep(10*time.Millisecond)
				wg.Add(1)
				go e.resolveDeadLockOne(executor, &wg)
			}
		}
	}
	e.order.Reset()
	wg.Wait()
}

func (e *Executor) resolveDeadLockOne(executor *executor.Executor, wg *sync.WaitGroup) {
	if executor == nil {
		wg.Done()
		return
	}
	var sql types.SQL
	if rand.Float64() < 0.5 {
		sql = types.SQL{
			SQLType: types.SQLTypeTxnCommit,
			SQLStmt: "COMMIT",
		}
	} else {
		sql = types.SQL{
			SQLType: types.SQLTypeTxnRollback,
			SQLStmt: "ROLLBACK",
		}
	}
	// exec commit/rollback
	executor.ExecSQL(&sql)
	// wait for txn finish
	go func() {
		for true {
			time.Sleep(time.Millisecond)
			if !executor.IfTxn() {
				wg.Done()
				break
			}
		}
	}()
}
