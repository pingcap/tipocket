package core

// Lock wrap mutex.Lock
func (e *Executor) Lock() {
	e.mutex.Lock()
	e.ifLock = true
}

// Unlock wrap mutex.Unlock
func (e *Executor) Unlock() {
	e.ifLock = false
	e.mutex.Unlock()
}
