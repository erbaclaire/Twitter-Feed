package feed

import (
	"math"
	"encoding/json"
	"src/lock"
)

// Feed represents a user's twitter feed
// You will add to this interface the implementations as you complete them.
type Feed interface {
	Add(body string, timestamp float64)
	Remove(timestamp float64) bool
	Contains(timestamp float64) bool
	ShowFeed() [][]byte
}

// feed is the internal representation of a user's twitter feed (hidden from outside packages)
// You CAN add to this structure but you cannot remove any of the original fields. You must use
// the original fields in your implementation. You can assume the feed will not have duplicate posts
type feed struct {
	start *post // a pointer to the beginning post
	lock   lock.RWMutex // a read-write lock on the feed - coarse grained
}

// post is the internal representation of a post on a user's twitter feed (hidden from outside packages)
// You CAN add to this structure but you cannot remove any of the original fields. You must use
// the original fields in your implementation.
type post struct {
	body      string // the text of the post
	timestamp float64  // Unix timestamp of the post
	next      *post  // the next post in the feed
	
}

// postBodyTimestamp is a structure that allows post data for FEED return in twitter.gp.
type postBodyTimestamp struct {
	Body      string 
	Timestamp float64	
}

// NewPost creates and returns a new post value given its body and timestamp
func newPost(body string, timestamp float64, next *post) *post {
	return &post{body, timestamp, next}
}

//NewFeed creates a empty user feed
func NewFeed() Feed {
	initFeed := newPost("null", math.Inf(-1), newPost("", math.Inf(1), nil))
	lock := lock.NewRWMutex()
	return &feed{start: initFeed, lock: lock}
}

// Add inserts a new post to the feed. The feed is always ordered by the timestamp where
// the most recent timestamp is at the beginning of the feed followed by the second most
// recent timestamp, etc. You may need to insert a new post somewhere in the feed because
// the given timestamp may not be the most recent.
// Implemented with coarse-grained locking.
func (f *feed) Add(body string, timestamp float64) {
	f.lock.Lock()

	pred := f.start
	curr := pred.next

	for (curr.timestamp < timestamp) {
		pred = curr
		curr = curr.next
	}
	
	newPost := newPost(body, timestamp, curr)
	pred.next = newPost

	f.lock.Unlock()
}

// Remove deletes the post with the given timestamp. If the timestamp
// is not included in a post of the feed then the feed remains
// unchanged. Return true if the deletion was a success, otherwise return false
// Implemented with coarse-grained locking
func (f *feed) Remove(timestamp float64) bool {
	f.lock.Lock()

	pred := f.start
	curr := pred.next

	for (curr.timestamp < timestamp) {
		pred = curr
		curr = curr.next
	}

	if curr.timestamp == timestamp {
		pred.next = curr.next
		f.lock.Unlock()
		return true
	}
	f.lock.Unlock()
	return false
}

// Contains determines whether a post with the given timestamp is
// inside a feed. The function returns true if there is a post
// with the timestamp, otherwise, false.
// Implemented with coarse-grained locking.
func (f *feed) Contains(timestamp float64) bool {
	f.lock.RLock()

	pred := f.start
	curr := pred.next

	for (curr.timestamp < timestamp) {
		pred = curr
		curr = curr.next
	}

	f.lock.RUnlock()

	return curr.timestamp == timestamp 
}

// reverseFeed reverses the posts to make the newest posts first.
func reverseFeed(input [][]byte) [][]byte {
    if len(input) == 0 {
        return input
    }
    return append(reverseFeed(input[1:]), input[0]) 
}

// ShowFeed puts post body and timestamp data in to byte data for FEED to return in twitter.go.
func (f *feed) ShowFeed() [][]byte {

	feedArray := make([][]byte, 0)
	f.lock.RLock()
	post := f.start.next
	for post.timestamp != math.Inf(1) {
		postByte, _ := json.Marshal(postBodyTimestamp{Body: post.body, Timestamp: post.timestamp})
		feedArray = append(feedArray, postByte)
		post = post.next
	}
	f.lock.RUnlock()
	// Reverse feed so that newest posts are first/
	return reverseFeed(feedArray)
}