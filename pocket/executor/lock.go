package executor

// import (
// 	"time"
// )

// // Lock wrap mutex.Lock and make marks
// func (e *Executor) Lock() {
// 	e.mutex.Lock()
// 	e.lockTime = time.Now()
// 	e.ifLock = true
// }

// // Unlock wrap mutex.Unlock and make marks
// func (e *Executor) Unlock() {
// 	e.lockTime = time.Time{}
// 	e.ifLock = false
// 	e.mutex.Unlock()
// }
