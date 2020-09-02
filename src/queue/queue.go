package queue

import (
	"encoding/json"
	"sync/atomic"
	"unsafe"
)

// Used "The Art of Multiprocessor Programming," pp. 230-233 for inspiration.
// Collaboration with Yifei on implementation.
// Help from Paul Zimnoch, a Software Engineer who implements locks for a living - just given clues not answers.

// Queue interface represents a non-blocking, unbounded queue with the following methods.
type Queue interface {
	Enqueue(byteTask []byte)
	Dequeue() []byte
}

// queue is the internal representation of the requests/tasks that need to be processed.
// It is initialized with a sentinel task as thge head and tail.
// This is a lock-free, unbounded queue.
type queue struct {
	head *task
	tail *task
}

// task is the internal representation of a request.
// It includes data represented by a slice of bytes and next which points to the next task.
type task struct {
	byteTask []byte
	next	 *task
}

// Data is used to unmarshall the JSON data when returning the sentinel node.
type Data struct {
	Command 	string 	`json:"command"`
	Id			int     `json:"id"`  
	Body        string  `json:"body"`
	Timestamp   float64 `json:"timestamp"`
	Value 		string  `json:"value"`
}

// newTask initializes a new task with byte data that represents the task and 
// a pointer to the next task.
// It is not publically accessible.
func newTask(byteTask []byte, next *task) *task {
    return &task{byteTask, next}
}

// NewQueue initializes a new empty queue with a sentinel value as the head and tail.
// The sentinel value's next value is nil
func NewQueue() *queue {
    q := new(queue)
    q.head = new(task)
    q.tail = q.head
	return q
}

// Enqueue adds a task to the end of the queue.
// The added task points to nil.
// The current tail points to the new task (done atomically) and the now previous tail
// points to the new tail (done non-atomically with updating the tail's next pointer).
// This is a lock-free implementation of enqueue.
func (q *queue) Enqueue(byteTask []byte) {
    var expectTail, expectTailNext *task
    newTask := newTask(byteTask, nil)

    success := false
    for !success {

        expectTail = q.tail
        expectTailNext = expectTail.next

        // If not at the tail then try again
        if q.tail != expectTail {
            continue
        }

        // If expected tail is not nil help it along and try again
        if expectTailNext != nil {
            atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&q.tail)), unsafe.Pointer(expectTail), unsafe.Pointer(expectTailNext))
            continue
        }
        
        // Logical enqueue
        success = atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&q.tail.next)), unsafe.Pointer(expectTailNext), unsafe.Pointer(newTask))
    }

    // Physical enqueue
    atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&q.tail)), unsafe.Pointer(expectTail), unsafe.Pointer(newTask))
}

// Dequeue removes a task from the head of the queue.
// The head then points to what the removed task pointed to.
// Dequeue returns the task that was dequeued from the head.
// If there are no tasks to dequeue, then the sentinel value is returned to indicate this to the calling routine. 
// Slight catch is that sometimes the head and tail point to the same task because the tail
// has updated the next pointer from the previous tail in enqueue but has not updated tail to be the new tail.
// When this happens the function "helps" the tail get to where it is supposed to be. If we did not do that
// then the tail pointer would be deleted and mess up the program.
// This is a lock-free implementation of dequeue.
func (q *queue) Dequeue() []byte {
    var dequeued []byte
    var expectSentinel, expectRemoved, expectTail *task

    success := false
    for !success {
        expectSentinel = q.head
        expectRemoved = expectSentinel.next
        expectTail = q.tail

        // If not at the head then try again
        if q.head != expectSentinel {
            continue 
        }

        // Signal that queue is empty when the sentinel node is reached
        if expectRemoved == nil {
            d, _ := json.Marshal(Data{Value: "sentinel"})
            return d 
        }

        // Help tail along if it is behind and try again
        if expectTail == expectSentinel {
            atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&q.tail)), unsafe.Pointer(expectTail), unsafe.Pointer(expectRemoved))
            continue
        }

        // Otherwise, dequeue and return the byte task
        dequeued = expectRemoved.byteTask
        success = atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&q.head)), unsafe.Pointer(expectSentinel), unsafe.Pointer(expectRemoved)) // dequeue
    }

    return dequeued

}