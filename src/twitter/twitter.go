package main

import (
	"os"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"src/queue"
	"src/feed"
	"encoding/json"
	"bufio"
)

func printUsage() {
	fmt.Println("Usage: twitter <number of goroutines> <block size>\n<number of goroutines> = the number of goroutines to be part of the queue\n<block size> = the maximum number of tasks a goroutine can process at any given point in time)")
}

// SharedContext houses variables shared by all goroutines.
type SharedContext struct {
	mutex            *sync.Mutex
	cond             *sync.Cond
	wg               *sync.WaitGroup
	numOfTasks       *int64 		// current number of tasks in the queue
	doneBool         *bool   	    // a boolean value to indicate if the DONE task has been read by the producer    
}

// ClientMessage represents the possible JSON input from the Client (producer tasks).
type ClientMessage struct {
	Command   	string  `json:"command"`
	Id		  	int     `json:"id"`  
	Body      	string  `json:"body,omitempty"`
	Timestamp 	float64 `json:"timestamp,omitempty"`
	Value 	  	string  `json:"value,omitempty"` // Value indicates if we have gotten to the sentinel value.
}

// ServerSuccessMessage represents the possible JSON response returned from the Server after completing an Add, Remove, or Contains task.
type ServerSuccessMessage struct {
	Success 	*bool           `json:"success"`
	Id      	int             `json:"id"` 
}

// ServerFeedMessage represents the JSON response returned from the Server after completing a Feed task.
type ServerFeedMessage struct {
	Id      	int             `json:"id"`
	Feed    	[]PostData      `json:"feed"`  
}

// PostData represents the JSON response for one Feed post.
type PostData struct {
	Body      	string  `json:"body"`
	Timestamp 	float64 `json:"timestamp"`
}

// addPostTask adds a post to the feed by calling the feed's Add method.
// A success message is printed to Stdout.
func addPostTask(feed feed.Feed, task ClientMessage) {
	feed.Add(task.Body, task.Timestamp)
	trueBool := true
	sm, _ := json.MarshalIndent(ServerSuccessMessage{Success: &trueBool, Id: task.Id}, "", "  ")	
	fmt.Printf("%s\n", sm)
}

// removePostTask removes a post frome the feed by calling the feed's Remove method.
// A success or failure message is printed to Stdout.
func removePostTask(feed feed.Feed, task ClientMessage) {
	removedBool := feed.Remove(task.Timestamp)
	sm, _ := json.MarshalIndent(ServerSuccessMessage{Success: &removedBool, Id: task.Id}, "", "   ")
	fmt.Printf("%s\n", sm)
}

// containsPostTask indicates if a feed contains a given post by calling the feed's Contains method.
// A success or failure message is printed to Stdout.
func containsPostTask(feed feed.Feed, task ClientMessage) {
	containsBool := feed.Contains(task.Timestamp)
	sm, _ := json.MarshalIndent(ServerSuccessMessage{Success: &containsBool, Id: task.Id}, "", "   ")
	fmt.Printf("%s\n", sm)
}

// showFeedTask prints to Stdout all the posts in a feed with the most recent post first.
// Each post displays the post's body and timestamp.
func showFeedTask(feed feed.Feed, task ClientMessage) {
	postByteArray := feed.ShowFeed()
	feedArray := []PostData{}
	for _, post := range(postByteArray) {
		var pd PostData
		err := json.Unmarshal(post, &pd)
		if err != nil {
			fmt.Println("error: ", err)
		}
		feedArray = append(feedArray, pd)
	}
	sm, _ := json.MarshalIndent(ServerFeedMessage{Id: task.Id, Feed: feedArray}, "", "   ")
	fmt.Printf("%s\n", sm)
}

// The consumer() function dequeues tasks and processes them.
// A goroutine will wait until there are tasks to process.
// Once there are tasks in the queue, a single goroutine is woken up to grab up to <block> amount
// of tasks and process those tasks.
// When the goroutine finishes those tasks it goes back to waiting for tasks to be added to the 
// queue with the other goroutines.
// When the DONE task is processed the remainder of tasks in the queue are processed and the goroutine returns.
func consumer(id int64, block int64, feed feed.Feed, queue queue.Queue, ctx *SharedContext) {
	// While there are more tasks
	for true{

		// Local flag for whether this should be this goroutine's last iteration.
		// It is always initially set to false and updated based on whether the DONE task has been read and if 
		// there are more tasks to process.
		exit := false

		// Lock to make sure we have an accurate read on the condition var.
		ctx.mutex.Lock()

		// Wait until there are tasks to consume.
		// If we have read the DONE task, though, just go because producer not signaling anymore.
		if atomic.LoadInt64(ctx.numOfTasks) == 0 && !*ctx.doneBool {
			ctx.cond.Wait()
		}

		ctx.mutex.Unlock() // Unlocks because dequeuing is done with a lock free queue so there should not be any issues with overlapping goroutines.

		// When you wake up grab block amount of tasks or all the tasks if there are < block amount.
		var blockOfTasks []ClientMessage
		for i := int64(0); i < block; i++ {
			var cm ClientMessage
			err := json.Unmarshal(queue.Dequeue(), &cm)
			if err != nil {
				fmt.Println("error: ", err)
				break
			}
			// If sentinel value is returned there are no more tasks to consume.
			if cm.Value == "sentinel" {
				break
			} else {
				blockOfTasks = append(blockOfTasks, cm)
				atomic.AddInt64(ctx.numOfTasks, -1) // Do this atomically as to not have to lock down the entire lock.
			}
		}

		// If there are no more tasks when the DONE task is read then the go routine exits when it completes its tasks.
		if *ctx.doneBool {
			if atomic.LoadInt64(ctx.numOfTasks) == 0 { 
				exit = true
			}
		}

		// Perform tasks
		if len(blockOfTasks) != 0 {
			for _, task := range(blockOfTasks) {
				if task.Command == "ADD" { // Add a post.
					addPostTask(feed, task)
				} else if task.Command == "REMOVE" { // Remove a post.
					removePostTask(feed, task)
				} else if task.Command == "CONTAINS" { // See if feed contains a post.
					containsPostTask(feed, task)
				} else if task.Command == "FEED" { // Visualize the feed.
					showFeedTask(feed, task)
				} 
			}
		}

		if exit {
			break
		}
	}

	ctx.wg.Done()
}

// producer reads in tasks from os.Stdin and adds these tasks to the queue.
// When a producers adds a task, if there are goroutines waiting on tasks to consume,
// the producer will wake one of these goroutine up to grab tasks.
func producer(queue queue.Queue, ctx *SharedContext) {

	// Read in tasks and add to the queue
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		task := scanner.Text()
		taskJSONBytes := []byte(task)
		var cm ClientMessage
		err := json.Unmarshal(taskJSONBytes, &cm)
		if err != nil {
			fmt.Println("error: ", err)
		}
		if cm.Command != "DONE" {	
			queue.Enqueue(taskJSONBytes)
			atomic.AddInt64(ctx.numOfTasks, 1) // Atomically adding so that the entire context does not need to be locked.
			ctx.cond.Signal() // Signal to a waiting task it can go.
		} else { // Stop producing if DONE task has been read.
			ctx.mutex.Lock()
			*ctx.doneBool = true
			ctx.cond.Broadcast() // Signal to waiting tasks they can go.
			ctx.mutex.Unlock()
			break
		}
	}
}

// main reads in the number of threads and the maximum number of tasks a given thread can process at once.
// main spawns goroutines to consume tasks and then calls producer to read in tasks for the consumers to
// consume. 
// main goroutine exits when all tasks in the queue are completed and the DONE task has been read.
func main() {

	// Create a new feed.
	feed := feed.NewFeed()

	// Initialize a new queue.
	queue := queue.NewQueue()

	// If command line arguments are not given, then run the tasks sequentially
	if len(os.Args) != 3 {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			task := scanner.Text()
			taskJSONBytes := []byte(task)
			var cm ClientMessage
			err := json.Unmarshal(taskJSONBytes, &cm)
			if err != nil {
				fmt.Println("error: ", err)
			}
			if cm.Command == "ADD" { // Add a post.
				addPostTask(feed, cm)
			} else if cm.Command == "REMOVE" { // Remove a post.
				removePostTask(feed, cm)
			} else if cm.Command == "CONTAINS" { // See if feed contains a post.
				containsPostTask(feed, cm)
			} else if cm.Command == "FEED" { // Visualize the feed.
				showFeedTask(feed, cm)
			} else if cm.Command == "DONE" { // Stop reading from stdin.
				break
			}
		}

	} else { // Otherwise spawn threads as consumers and produce tasks to queue

		// Read in command line arguments.
		threads, _ := strconv.ParseInt(os.Args[1], 10, 64)
		block, _ := strconv.ParseInt(os.Args[2], 10, 64)

		// Initialize sync mechanisms.
		var wg            sync.WaitGroup
		var mtx           sync.Mutex
		var numOfTasks    int64
		doneBool := false

		condVar := sync.NewCond(&mtx)
		context := SharedContext{wg: &wg, cond: condVar, mutex: &mtx, numOfTasks: &numOfTasks, doneBool: &doneBool}

		// Spawn goroutines
		for i := int64(0); i < threads; i++ {
			wg.Add(1)
			go consumer(i, block, feed, queue, &context)
		}

		// Start producing tasks.
		producer(queue, &context)

		wg.Wait()


	}
}
