// Package lock provides an implementation of a read-write lock
// that uses condition variables and mutexes.
package lock

import (
	"sync"
)

// RWMutex represents the functionality of a Read-Write lock.
type RWMutex interface {
	Lock()
	Unlock()
	RLock()
	RUnlock()
}

// rwmutex  is an internal representation of a Read-Write lock. It is not accessible
// to outside programs.
type rwmutex struct {
	cond       	 *sync.Cond	// sync.Cond has a mutex in it
	readCount  	 int	
}

// NewRWMutex initializes a new Read-Write lock with a conditional synchronization
// mechanism and a readCount which is initialized to 0 to keep track of the number
// of readers currently reading the data.
func NewRWMutex() *rwmutex {
	condVar := sync.NewCond(new(sync.Mutex))
	var readCount int
	return &rwmutex{condVar, readCount} 
}

// Lock locks rw for writing. If the lock is already locked for reading or writing
// indicated by the readCount being greater than 0, the lock blocks until the lock 
// is available (i.e. there are no more goroutines reading). 
func (rw *rwmutex) Lock() {
	rw.cond.L.Lock()
	for rw.readCount != 0 {
		rw.cond.Wait()
	}
}

// Unlock unlocks rw for writing. It is a run-time error if rw is not locked for
// writing on entry to Unlock. It signals waiting readers and writers that it is 
// done and they can wake up.
func (rw *rwmutex) Unlock() {
	rw.cond.Signal()
	rw.cond.L.Unlock()
}

// RLock locks for reading. It should not be used for recursive read locking. RLock
// first locks the mutex when it can and checks that there are not more than 64 readers
// already. If there are, the thread must Wait until that is no longer true. Then 
// the readCount is incremented and the mutex unlocked so that other readers can read at
// the same time.
func (rw *rwmutex) RLock() {
	rw.cond.L.Lock()
	for rw.readCount > 64 {
		rw.cond.Wait()
	}
	rw.readCount++
	rw.cond.L.Unlock()
}

// Unlock unlocks rw for reading. It is a run-time error if rw is not locked for
// reading on entry to RUnlock. RUnlock first locks the mutex and decrements the
// readCount. It signals if the count is now equal to 0 such that any waiting
// writer could try to acquire the lock. It also wakes up any reader waiting on
// the readCount to be no more than 64. It then unlocks the mutex.
func (rw *rwmutex) RUnlock() {
	rw.cond.L.Lock()
	rw.readCount--
	if rw.readCount == 0 {
		rw.cond.Signal()
	}
	if rw.readCount <= 64 { // Even if a sleeping writer is signaled with this and readCount > 0, the writer
		rw.cond.Signal()    // will go back to sleep because it is in a for-loop checking readCount.
	}                       
	rw.cond.L.Unlock()
}