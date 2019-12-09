package core

import (
	"math/rand"
	"time"
	// "github.com/ngaut/log"
	"github.com/pingcap/tipocket/pocket/executor"
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
				// log.Info("deadlock detected")
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
	// log.Infof("last execute ID is %d\n", lastExecID)
	// var lastResolveExecutor *executor.Executor
	// for _, executor := range e.executors {
	// 	if executor.GetID() == lastExecID {
	// 		lastResolveExecutor = executor
	// 		continue
	// 	}
	// 	e.resolveDeadLockOne(executor)
	// }
	// e.resolveDeadLockOne(lastResolveExecutor)
	for e.order.Next() {
		for _, executor := range e.executors {
			if executor.GetID() == e.order.Val() {
				time.Sleep(2*time.Millisecond)
				e.resolveDeadLockOne(executor)
				time.Sleep(2*time.Millisecond)
			}
		}
	}
}

func (e *Executor) resolveDeadLockOne(executor *executor.Executor) {
	if executor == nil {
		return
	}
	// log.Infof("resolve lock on executor-%d, ch len %d\n", executor.GetID(), len(executor.TxnReadyCh))
	if rand.Float64() < 0.5 {
		// log.Info("commit", len(executor.TxnReadyCh))
		_ = executor.TxnCommit()
	} else {
		// log.Info("rollback", len(executor.TxnReadyCh))
		// _ = executor.TxnCommit()
		_ = executor.TxnRollback()
	}
	// log.Infof("resolve lock done executor-%d, ch len %d\n", executor.GetID(), len(executor.TxnReadyCh))
}
