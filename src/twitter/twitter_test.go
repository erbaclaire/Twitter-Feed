package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os/exec"
	"strconv"
	"testing"
	"time"
)

type _TestAddRequest struct {
	Command   	string 	`json:"command"`
	Id			int64 	`json:"id"`
	Timestamp	float64	`json:"timestamp"`
	Body      	string	`json:"body"`
}
type _TestRemoveRequest struct {
	Command   	string 	`json:"command"`
	Id			int64 	`json:"id"`
	Timestamp	float64	`json:"timestamp"`
}
type _TestContainsRequest struct {
	Command   	string 	`json:"command"`
	Id			int64 	`json:"id"`
	Timestamp	float64	`json:"timestamp"`
}
type _TestFeedRequest struct {
	Command   	string 	`json:"command"`
	Id			int64 	`json:"id"`
}
type _TestDoneRequest struct {
	Command 	string `json:"command"`
}

type _TestNormalResponse struct {
	Success     bool       `json:"success"`
	Id 			int64	   `json:"id"`
}

type _TestFeedResponse struct {
	Id 			int64	    `json:"id"`
	Feed 		[]_TestPostData	`json:"feed"`
}

type _TestPostData struct {
	Body		string	`json:"body"`
	Timestamp	float64	`json:"timestamp"`
}

func generateSlice(size int) []int {

	slice := make([]int, size, size)
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < size; i++ {
		slice[i] = i
	}
	return slice
}
func createAdds(numbers []int, idx int) (map[int]_TestAddRequest, map[int]_TestNormalResponse, int) {

	requests := make(map[int]_TestAddRequest)
	responses := make(map[int]_TestNormalResponse)

	for _, number := range numbers {
		numberStr := strconv.Itoa(number)
		request := _TestAddRequest{"ADD",int64(idx), float64(number),numberStr}
		response := _TestNormalResponse{true,int64(idx)}
 		requests[idx] = request
 		responses[idx] = response
 		idx++
	}
	return requests, responses, idx
}
func createContains(numbers []int, successes []bool, idx int ) (map[int]_TestContainsRequest, map[int]_TestNormalResponse, int) {

	requests := make(map[int]_TestContainsRequest)
	responses := make(map[int]_TestNormalResponse)

	for i, number := range numbers {
		request := _TestContainsRequest{"CONTAINS",int64(idx), float64(number)}
		response := _TestNormalResponse{successes[i],int64(idx)}
		requests[idx] = request
		responses[idx] = response
		idx++
	}
	return requests, responses, idx
}
func createRemoves(numbers []int, successes []bool, idx int) (map[int]_TestRemoveRequest, map[int]_TestNormalResponse, int) {

	requests := make(map[int]_TestRemoveRequest)
	responses := make(map[int]_TestNormalResponse)

	for i, number := range numbers {
		request := _TestRemoveRequest{"REMOVE",int64(idx), float64(number)}
		response := _TestNormalResponse{successes[i],int64(idx)}
		requests[idx] = request
		responses[idx] = response
		idx++
	}
	return requests, responses, idx
}
func createFeed(numbers []int, idx int) (_TestFeedRequest, _TestFeedResponse, int) {


    postData := make([]_TestPostData,len(numbers))
	request := _TestFeedRequest{"FEED",int64(idx)}

	for i, number := range numbers {
		numberStr := strconv.Itoa(number)
		postData[i] = _TestPostData{numberStr,float64(number)}
	}
	response := _TestFeedResponse{int64(idx), postData}
	return request, response, idx + 1
}


// This test only provides a "DONE" Command, which should cause the program to exit immediately.
func TestSimpleDone(t *testing.T) {

	numOfThreadsStr := "4"
	blockSizeStr := "1"

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go",  "run" , "twitter.go", numOfThreadsStr, blockSizeStr)

	stdin, errIn := cmd.StdinPipe()
	if errIn != nil {
		t.Fatal("<runTwitter>: Error in Getting stdin pipe: Contact Professor Samuels, if see this message.")
	}

	if err := cmd.Start(); err != nil {
		t.Fatal("<cmd.Start> Error in Executing Test: Contact Professor Samuels, if see this message.")
	}
	done := make(chan bool)

	go func() {
		//decoder := json.NewDecoder(stdout)
		encoder := json.NewEncoder(stdin)

		request  := _TestDoneRequest{"DONE"}

		if err := encoder.Encode(&request); err != nil {
			t.Fatal("<cmd.encode> Error in Executing Test: Contact Professor Samuels, if see this message.")
		}
		done <- true
	}()

	<-done
	if err := cmd.Wait(); err != nil {
		t.Errorf("The automated test timed out. You may have a deadlock, starvation issue and/or you did not implement" +
			" the necessary code for passing this test.")
	}
}

// This test only provides a "DONE" Command, it waits a few seconds before sending the command.
// It then should cause the program to exit immediately.
func TestSimpleWaitDone(t *testing.T) {

	numOfThreadsStr := "4"
	blockSizeStr := "1"

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go",  "run" , "twitter.go", numOfThreadsStr, blockSizeStr)
	/*stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal("<runTwitter>: Error in Getting stdout pipe: Contact Professor Samuels, if see this message.")
	}*/
	stdin, errIn := cmd.StdinPipe()
	if errIn != nil {
		t.Fatal("<runTwitter>: Error in Getting stdin pipe: Contact Professor Samuels, if see this message.")
	}

	if err := cmd.Start(); err != nil {
		t.Fatal("<cmd.Start> Error in Executing Test: Contact Professor Samuels, if see this message.")
	}
	done := make(chan bool)

	go func() {
		encoder := json.NewEncoder(stdin)

		request  := _TestDoneRequest{"DONE"}

		time.Sleep(3*time.Second)
		if err := encoder.Encode(&request); err != nil {
			t.Fatal("<cmd.encode> Error in Executing Test: Contact Professor Samuels, if see this message.")
		}
		done <- true
	}()

	<-done
	if err := cmd.Wait(); err != nil {
		t.Errorf("The automated test timed out. You may have a deadlock, starvation issue and/or you did not implement" +
			" the necessary code for passing this test.")
	}
}

// This test only provides "ADD" commands, it checks that we get back the right acknowledgements and makes the program
// wait 30 seconds before continuing.
func TestAddRequests(t *testing.T) {

	numOfThreadsStr := "3"
	blockSizeStr := "10"

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go",  "run" , "twitter.go", numOfThreadsStr, blockSizeStr)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal("<runTwitter>: Error in Getting stdout pipe: Contact Professor Samuels, if see this message.")
	}
	stdin, errIn := cmd.StdinPipe()
	if errIn != nil {
		t.Fatal("<runTwitter>: Error in Getting stdin pipe: Contact Professor Samuels, if see this message.")
	}

	if err := cmd.Start(); err != nil {
		t.Fatal("<cmd.Start> Error in Executing Test: Contact Professor Samuels, if see this message.")
	}
	inDone := make(chan bool)
	outDone := make(chan bool)

	doneRequest  := _TestDoneRequest{"DONE"}
	numbers := generateSlice(100)
	requests, responses, _ := createAdds(numbers,0)

	go func() {
		encoder := json.NewEncoder(stdin)
		for _, request := range requests {
			if err := encoder.Encode(&request); err != nil {
				t.Fatal("<cmd.encode> Error in Executing Test: Contact Professor Samuels, if see this message.")
			}
		}
		if err := encoder.Encode(&doneRequest); err != nil {
			t.Fatal("<cmd.encode> Error in Executing Test: Contact Professor Samuels, if see this message.")
		}
		inDone <- true
	}()

	go func() {
		decoder := json.NewDecoder(stdout)
		var count int
		for {
			var response _TestNormalResponse
			if err := decoder.Decode(&response); err != nil {
				break
			}
			if value, ok := responses[int(response.Id)]; ok {
				if value.Id != response.Id || value.Success != response.Success {
					t.Errorf("Id and Success fields do not match. Got(%v,%v), Expected(%v,%v)",
						response.Id, response.Success, value.Id, value.Success)
				}
				count++
			} else {
				t.Errorf("Received an invalid id back from twitter.go. We only We should only have ids between 0-99 but got:%v", response.Id)
			}
			if count % 5 == 0 {
				time.Sleep(1 * time.Second)
			}
		}
		if count != 100 {
			t.Errorf("Did not receieve the right amount of Add acknowledgements. Got:%v, Expected:%v", count, len(requests))
		}
		outDone <- true
	}()

	<-inDone
	<-outDone
	if err := cmd.Wait(); err != nil {
		t.Errorf("The automated test timed out. You may have a deadlock, starvation issue and/or you did not implement" +
			" the necessary code for passing this test.")
	}
}
// This test only requests a single ADD request and then exits but the tests has a high number of threads and
// block size spawned.
func TestSimpleAddRequest(t *testing.T) {

	numOfThreadsStr := "16"
	blockSizeStr := "100"

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go",  "run" , "twitter.go", numOfThreadsStr, blockSizeStr)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal("<runTwitter>: Error in Getting stdout pipe: Contact Professor Samuels, if see this message.")
	}
	stdin, errIn := cmd.StdinPipe()
	if errIn != nil {
		t.Fatal("<runTwitter>: Error in Getting stdin pipe: Contact Professor Samuels, if see this message.")
	}

	if err := cmd.Start(); err != nil {
		t.Fatal("<cmd.Start> Error in Executing Test: Contact Professor Samuels, if see this message.")
	}
	inDone := make(chan bool)
	outDone := make(chan bool)

	doneRequest  := _TestDoneRequest{"DONE"}
	numbers := generateSlice(1)
	requests, responses, _ := createAdds(numbers,0)

	go func() {
		encoder := json.NewEncoder(stdin)
		for _, request := range requests {
			time.Sleep(1 * time.Second) // Wait a second before sending the add request
			if err := encoder.Encode(&request); err != nil {
				t.Fatal("<cmd.encode> Error in Executing Test: Contact Professor Samuels, if you see this message.")
			}
		}
		if err := encoder.Encode(&doneRequest); err != nil {
			t.Fatal("<cmd.encode> Error in Executing Test: Contact Professor Samuels, if you see this message.")
		}
		inDone <- true
	}()

	go func() {
		decoder := json.NewDecoder(stdout)
		var count int
		for {
			var response _TestNormalResponse
			if err := decoder.Decode(&response); err != nil {
				break
			}
			if value, ok := responses[int(response.Id)]; ok {
				if value.Id != response.Id || value.Success != response.Success {
					t.Errorf("Id and Success fields do not match. Got(%v,%v), Expected(%v,%v)",
						response.Id, response.Success, value.Id, value.Success)
				}
				count++
			} else {
				t.Errorf("Received an invalid id back from twitter.go. We only Added the number 'O' but got(id):%v, expected(id):0", response.Id)
			}
			if count % 5 == 0 {
				time.Sleep(1 * time.Second)
			}
		}
		if count != 1 {
			t.Errorf("Did not receieve the right amount of Add acknowledgements. Got:%v, Expected:%v", count, len(requests))
		}
		outDone <- true
	}()

	<-inDone
	<-outDone
	if err := cmd.Wait(); err != nil {
		t.Errorf("The automated test timed out. You may have a deadlock, starvation issue and/or you did not implement" +
			" the necessary code for passing this test.")
	}
}


// This test only performs a few contains requests before adding anything to the feed, therefore they all should return
// False.
func TestSimpleContainsRequest(t *testing.T) {

	numOfThreadsStr := "16"
	blockSizeStr := "1"

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go",  "run" , "twitter.go", numOfThreadsStr, blockSizeStr)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal("<runTwitter>: Error in Getting stdout pipe: Contact Professor Samuels, if see this message.")
	}
	stdin, errIn := cmd.StdinPipe()
	if errIn != nil {
		t.Fatal("<runTwitter>: Error in Getting stdin pipe: Contact Professor Samuels, if see this message.")
	}

	if err := cmd.Start(); err != nil {
		t.Fatal("<cmd.Start> Error in Executing Test: Contact Professor Samuels, if see this message.")
	}
	inDone := make(chan bool)
	outDone := make(chan bool)

	doneRequest  := _TestDoneRequest{"DONE"}
	successSlice :=  make([]bool, 30)
	numbers := generateSlice(30)
	requests, responses, _ := createContains(numbers,successSlice, 0)

	go func() {
		encoder := json.NewEncoder(stdin)
		for _, request := range requests {
			time.Sleep(1 * time.Millisecond) // Wait a second before sending the add request
			if err := encoder.Encode(&request); err != nil {
				t.Fatal("<cmd.encode> Error in Executing Test: Contact Professor Samuels, if see this message.")
			}
		}
		if err := encoder.Encode(&doneRequest); err != nil {
			t.Fatal("<cmd.encode> Error in Executing Test: Contact Professor Samuels, if see this message.")
		}
		inDone <- true
	}()

	go func() {
		decoder := json.NewDecoder(stdout)
		var count int
		for {
			var response _TestNormalResponse
			if err := decoder.Decode(&response); err != nil {
				break
			}
			if value, ok := responses[int(response.Id)]; ok {
				if value.Id != response.Id || value.Success != response.Success {
					t.Errorf("Id and Success fields do not match. Got(%v,%v), Expected(%v,%v)",
						response.Id, response.Success, value.Id, value.Success)
				}
				count++
			} else {
				t.Errorf("Received an invalid id back from twitter.go. We only Added the number 'O' but got(id):%v, expected(id):0", response.Id)
			}
			if count % 5 == 0 {
				time.Sleep(1 * time.Second)
			}
		}
		if count != 30 {
			t.Errorf("Did not receieve the right amount of Add acknowledgements. Got:%v, Expected:%v", count, len(requests))
		}
		outDone <- true
	}()

	<-inDone
	<-outDone
	if err := cmd.Wait(); err != nil {
		t.Errorf("The automated test timed out. You may have a deadlock, starvation issue and/or you did not implement" +
			" the necessary code for passing this test.")
	}
}
// This test Adds the numbers [0-30] into the feed but also throws in some contains requests. We don't care about the
// contains requests but rather just ensuring that they get returned by the twitter.go
func TestAddWithContainsRequest(t *testing.T) {

	numOfThreadsStr := "4"
	blockSizeStr := "3"

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go",  "run" , "twitter.go", numOfThreadsStr, blockSizeStr)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal("<runTwitter>: Error in Getting stdout pipe: Contact Professor Samuels, if see this message.")
	}
	stdin, errIn := cmd.StdinPipe()
	if errIn != nil {
		t.Fatal("<runTwitter>: Error in Getting stdin pipe: Contact Professor Samuels, if see this message.")
	}

	if err := cmd.Start(); err != nil {
		t.Fatal("<cmd.Start> Error in Executing Test: Contact Professor Samuels, if see this message.")
	}
	inDone := make(chan bool)
	outDone := make(chan bool)

	doneRequest  := _TestDoneRequest{"DONE"}
	successSlice :=  make([]bool, 30)
	numbers := generateSlice(30)
	requestsAdd, responsesAdd, addIdx := createAdds(numbers, 0)
	requestsContains, responsesContains, _ := createContains(numbers,successSlice, addIdx)

	go func() {
		encoder := json.NewEncoder(stdin)
		for idx := 0; idx < len(requestsAdd); idx++ {
			requestAdd := requestsAdd[idx]
			if err := encoder.Encode(&requestAdd); err != nil {
				t.Fatal("<cmd.encode> Error in Executing Test: Contact Professor Samuels, if see this message.")
			}
			time.Sleep(1 * time.Millisecond) // Wait milisecond before sending the add request
			requestContains  := requestsContains[addIdx]
			if err := encoder.Encode(&requestContains); err != nil {
				t.Fatal("<cmd.encode> Error in Executing Test: Contact Professor Samuels, if see this message.")
			}
			addIdx++
		}
		if err := encoder.Encode(&doneRequest); err != nil {
			t.Fatal("<cmd.encode> Error in Executing Test: Contact Professor Samuels, if see this message.")
		}
		inDone <- true
	}()

	go func() {
		decoder := json.NewDecoder(stdout)
		var count int
		for {
			var response _TestNormalResponse
			if err := decoder.Decode(&response); err != nil {
				break
			}
			if value, ok := responsesAdd[int(response.Id)]; ok {
				if value.Id != response.Id || value.Success != response.Success {
					t.Errorf("Add Request & Response Id and Success fields do not match. Got(%v,%v), Expected(%v,%v)",
						response.Id, response.Success, value.Id, value.Success)
				}
				count++
			} else if value, ok := responsesContains[int(response.Id)]; ok {
				if value.Id != response.Id {
					t.Errorf("Contains Request & Response Id fields do not match. Got(%v), Expected(%v)",
						response.Id, value.Id)
				}
				count++
			} else {
				t.Errorf("Received an invalid id back from twitter.go. We only added ids [0,30] but got(id):%v", response.Id)
			}
			if count % 5 == 0 {
				time.Sleep(1 * time.Second)
			}
		}
		if count != len(requestsAdd) + len(requestsContains) {
			t.Errorf("Did not receieve the right amount of Add&Contains acknowledgements. Got:%v, Expected:%v", count, len(requestsAdd) + len(requestsContains))
		}
		outDone <- true
	}()

	<-inDone
	<-outDone
	if err := cmd.Wait(); err != nil {
		t.Errorf("The automated test timed out. You may have a deadlock, starvation issue and/or you did not implement" +
			" the necessary code for passing this test.")
	}
}
// This test checks to make sure the feed request returns empty on an empty feed.
func TestSimpleFeedRequest(t *testing.T) {
	numOfThreadsStr := "16"
	blockSizeStr := "100"

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go",  "run" , "twitter.go", numOfThreadsStr, blockSizeStr)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal("<runTwitter>: Error in Getting stdout pipe: Contact Professor Samuels, if see this message.")
	}
	stdin, errIn := cmd.StdinPipe()
	if errIn != nil {
		t.Fatal("<runTwitter>: Error in Getting stdin pipe: Contact Professor Samuels, if see this message.")
	}

	if err := cmd.Start(); err != nil {
		t.Fatal("<cmd.Start> Error in Executing Test: Contact Professor Samuels, if see this message.")
	}
	inDone := make(chan bool)
	outDone := make(chan bool)

	doneRequest  := _TestDoneRequest{"DONE"}

	numbers := []int{}
	request, responseExpected, _ := createFeed(numbers,0)

	go func() {
		encoder := json.NewEncoder(stdin)
		time.Sleep(1 * time.Millisecond) // Wait a second before sending the feed request
		if err := encoder.Encode(&request); err != nil {
			t.Fatal("<cmd.encode> Error in Executing Test: Contact Professor Samuels, if see this message.")
		}
		time.Sleep(1 * time.Millisecond) // Wait a second before sending the add request
		if err := encoder.Encode(&doneRequest); err != nil {
			t.Fatal("<cmd.encode> Error in Executing Test: Contact Professor Samuels, if see this message.")
		}
		inDone <- true
	}()

	go func() {
		decoder := json.NewDecoder(stdout)
		var count int
		for {
			var response _TestFeedResponse
			if err := decoder.Decode(&response); err != nil {
				break
			}
			if response.Id != responseExpected.Id  {
					t.Errorf("Feed Request & Response Id fields do not match. Got(%v), Expected(%v)",
						response.Id, responseExpected.Id)
			} else {
				if len(response.Feed) != 0 {
					t.Errorf("Feed Request was sent on an empty feed but the response return posts. Got(%v), Expected(%v)",
						len(response.Feed), 0)
				}
				count++
			}
			if count % 5 == 0 {
				time.Sleep(1 * time.Second)
			}
		}
		if count != 1 {
			t.Errorf("Did not receieve the right amount of Feed Request acknowledgements. Got:%v, Expected:%v", count, 1)
		}
		outDone <- true
	}()

	<-inDone
	<-outDone
	if err := cmd.Wait(); err != nil {
		t.Errorf("The automated test timed out. You may have a deadlock, starvation issue and/or you did not implement" +
			" the necessary code for passing this test.")
	}
}
//This function adds a random number of items to the feed and checks to make sure the feed request
// returns them in the right order based on timestamp.
func TestSimpleAddAndFeedRequest(t *testing.T) {
	numOfThreadsStr := "3"
	blockSizeStr := "1"

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go",  "run" , "twitter.go", numOfThreadsStr, blockSizeStr)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal("<runTwitter>: Error in Getting stdout pipe: Contact Professor Samuels, if see this message.")
	}
	stdin, errIn := cmd.StdinPipe()
	if errIn != nil {
		t.Fatal("<runTwitter>: Error in Getting stdin pipe: Contact Professor Samuels, if see this message.")
	}

	if err := cmd.Start(); err != nil {
		t.Fatal("<cmd.Start> Error in Executing Test: Contact Professor Samuels, if see this message.")
	}
	inDone := make(chan bool)
	outDone := make(chan bool)
	addDone := make(chan bool)

	doneRequest  := _TestDoneRequest{"DONE"}
	postInfo := []int{1, 2, 18, 9, 8, 20, 16, 10, 6, 14, 17, 15, 19, 5, 13, 11, 7, 4, 3, 12}
	requestsAdd, responsesAdd, addIdx := createAdds(postInfo, 0)
	order := []int{20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1}
	requestFeed, responseFeedExpected, _ := createFeed(order, addIdx)

	go func() {
		encoder := json.NewEncoder(stdin)
		for idx := 0; idx < len(postInfo); idx++ {
			requestAdd := requestsAdd[idx]
			if err := encoder.Encode(&requestAdd); err != nil {
				t.Fatal("<cmd.encode> Error in Executing Test: Contact Professor Samuels, if see this message.")
			}
		}
		<-addDone
		if err := encoder.Encode(&requestFeed); err != nil {
			t.Fatal("<cmd.encode> Error in Executing Test: Contact Professor Samuels, if see this message.")
		}
		if err := encoder.Encode(&doneRequest); err != nil {
			t.Fatal("<cmd.encode> Error in Executing Test: Contact Professor Samuels, if see this message.")
		}
		inDone <- true
	}()

	go func() {
		decoder := json.NewDecoder(stdout)
		var count int
		for {
			var response _TestNormalResponse
			if err := decoder.Decode(&response); err != nil {
				break
			}
			if value, ok := responsesAdd[int(response.Id)]; ok {
				if value.Id != response.Id || value.Success != response.Success {
					t.Errorf("Add Request & Response Id and Success fields do not match. Got(%v,%v), Expected(%v,%v)",
						response.Id, response.Success, value.Id, value.Success)
				}
				count++
			} else {
				t.Errorf("Received an invalid id back from twitter.go. We only added ids [0,30] but got(id):%v", response.Id)
			}
			if count == 20 {
				addDone <- true
				var responseFeed _TestFeedResponse
				if err := decoder.Decode(&responseFeed); err != nil {
					break
				}
				if responseFeed.Id != responseFeedExpected.Id  {
					t.Errorf("Feed Request & Response Id fields do not match. Got(%v), Expected(%v)",
						responseFeed.Id, responseFeedExpected.Id)
				} else {
					if len(responseFeedExpected.Feed) != len(responseFeed.Feed) {
						t.Errorf("Feed Response number of posts not equal to each other. Got(%v), Expected(%v)",
							len(responseFeed.Feed), len(responseFeedExpected.Feed))
					}
					for idx, post := range responseFeed.Feed {
						if post.Body != responseFeedExpected.Feed[idx].Body || post.Timestamp != responseFeedExpected.Feed[idx].Timestamp {
							t.Errorf("Feed Response Post Data does not match. This is checking that the order returned is correct. Got(Body:%v, TimeStamp:%v), Expected(Body:%v, TimeStamp:%v)",
								post.Body, post.Timestamp,  responseFeedExpected.Feed[idx].Body, responseFeedExpected.Feed[idx].Timestamp)
						}
					}
					count++
				}
			}
		}
		if count != len(requestsAdd) + 1 {
			t.Errorf("Did not receieve the right amount of Add&Feed acknowledgements. Got:%v, Expected:%v", count, len(requestsAdd) + 1)
		}
		outDone <- true
	}()

	<-inDone
	<-outDone
	if err := cmd.Wait(); err != nil {
		t.Errorf("The automated test timed out. You may have a deadlock, starvation issue and/or you did not implement" +
			" the necessary code for passing this test.")
	}
}
// This test tries to remove posts from an empty feed
func TestSimpleRemoveRequest(t *testing.T) {

	numOfThreadsStr := "4"
	blockSizeStr := "3"

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go",  "run" , "twitter.go", numOfThreadsStr, blockSizeStr)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal("<runTwitter>: Error in Getting stdout pipe: Contact Professor Samuels, if see this message.")
	}
	stdin, errIn := cmd.StdinPipe()
	if errIn != nil {
		t.Fatal("<runTwitter>: Error in Getting stdin pipe: Contact Professor Samuels, if see this message.")
	}

	if err := cmd.Start(); err != nil {
		t.Fatal("<cmd.Start> Error in Executing Test: Contact Professor Samuels, if see this message.")
	}
	inDone := make(chan bool)
	outDone := make(chan bool)

	doneRequest  := _TestDoneRequest{"DONE"}
	successSlice :=  make([]bool, 30)
	numbers := generateSlice(30)
	requestsRemoves, responsesRemoves, _ := createRemoves(numbers,successSlice, 0)

	go func() {
		encoder := json.NewEncoder(stdin)
		for idx := 0; idx < len(numbers); idx++ {
			requestRemove := requestsRemoves[idx]
			if err := encoder.Encode(&requestRemove); err != nil {
				t.Fatal("<cmd.encode> Error in Executing Test: Contact Professor Samuels, if see this message.")
			}
			time.Sleep(1 * time.Millisecond) // Wait milisecond before sending the add request
		}
		if err := encoder.Encode(&doneRequest); err != nil {
			t.Fatal("<cmd.encode> Error in Executing Test: Contact Professor Samuels, if see this message.")
		}
		inDone <- true
	}()

	go func() {
		decoder := json.NewDecoder(stdout)
		var count int
		for {
			var response _TestNormalResponse
			if err := decoder.Decode(&response); err != nil {
				break
			}
			if value, ok := responsesRemoves[int(response.Id)]; ok {
				if value.Id != response.Id || value.Success != response.Success {
					t.Errorf("Remove Request & Response Id and Success fields do not match. Got(%v,%v), Expected(%v,%v)",
						response.Id, response.Success, value.Id, value.Success)
				}
				count++
			} else {
				t.Errorf("Received an invalid id back from twitter.go. We only added ids [0,30] but got(id):%v", response.Id)
			}
			if count % 5 == 0 {
				time.Sleep(1 * time.Second)
			}
		}
		if count != len(responsesRemoves) {
			t.Errorf("Did not receieve the right amount of Remove acknowledgements. Got:%v, Expected:%v", count, len(responsesRemoves))
		}
		outDone <- true
	}()

	<-inDone
	<-outDone
	if err := cmd.Wait(); err != nil {
		t.Errorf("The automated test timed out. You may have a deadlock, starvation issue and/or you did not implement" +
			" the necessary code for passing this test.")
	}
}
// This test Adds the numbers [0-1000] into the feed but also throws in some contains requests. We don't care about the
// Contains requests but rather just ensuring that they get returned by the twitter.go. Then we remove the olds numbers
// and check to make sure the evens are still there and all the odds are gone.
func TestAllRequests(t *testing.T) {
	numOfThreadsStr := "40"
	blockSizeStr := "1"

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go",  "run" , "twitter.go", numOfThreadsStr, blockSizeStr)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal("<runTwitter>: Error in Getting stdout pipe: Contact Professor Samuels, if see this message.")
	}
	stdin, errIn := cmd.StdinPipe()
	if errIn != nil {
		t.Fatal("<runTwitter>: Error in Getting stdin pipe: Contact Professor Samuels, if see this message.")
	}

	if err := cmd.Start(); err != nil {
		t.Fatal("<cmd.Start> Error in Executing Test: Contact Professor Samuels, if see this message.")
	}
	inDone := make(chan bool)
	outDone := make(chan bool)

	/***** First Wave: Add all Posts and Random Contains ***/
	wave1Done := make(chan bool)
	doneRequest  := _TestDoneRequest{"DONE"}
	postInfo := []int{1, 2, 18, 9, 8, 20, 16, 10, 6, 14, 17, 15, 19, 5, 13, 11, 7, 4, 3, 12}
	requestsAdd, responsesAdd, addIdx := createAdds(postInfo, 0)
	successSlice :=  make([]bool, len(postInfo))
	requestsContains, responsesContains, containsIdx := createContains(postInfo,successSlice, addIdx)


	/**** Second Wave: Remove all the Even Numbers *****/
	wave2Done := make(chan bool)
	removePosts := []int{2,4,6,8,10,12,14,16,18,20}
	successSliceRemove :=  make([]bool, len(removePosts))
	for idx, _ := range removePosts {
		successSliceRemove[idx] = true
	}
	requestsRemoves, responsesRemoves, removeIdx := createRemoves(removePosts,successSliceRemove, containsIdx)

	/**** Third Wave: Removal all the odds and check the feed is empty **/
	wave3Done := make(chan bool)
	containsOrder := []int{2,4,6,8,10,12,14,16,18,20}
	successSliceContains :=  make([]bool, len(containsOrder))
	requestsContains2, responsesContains2, containsIdx2 := createContains(containsOrder,successSliceContains, removeIdx)

	removePosts2 := []int{19, 17, 15, 13, 11, 9, 7, 5, 3, 1}
	successSliceRemove2 :=  make([]bool, len(removePosts2))
	for idx, _ := range removePosts2 {
		successSliceRemove2[idx] = true
	}
	requestsRemoves2, responsesRemoves2, removeIdx2 := createRemoves(removePosts2,successSliceRemove2, containsIdx2)

	/**** Fourth Wave: check the feed is empty **/
	wave4Done := make(chan bool)
	order2 := []int{}
	requestFeed, responseFeedExpected, _ := createFeed(order2, removeIdx2)

	go func() {
		encoder := json.NewEncoder(stdin)
		for idx := 0; idx < len(postInfo); idx++ {
			requestAdd := requestsAdd[idx]
			if err := encoder.Encode(&requestAdd); err != nil {
				t.Fatal("<cmd.encode> Error in Executing Test: Contact Professor Samuels, if see this message.")
			}
		}
		for idx := addIdx; idx < (addIdx + 20); idx++ {
			requestContains := requestsContains[idx]
			if err := encoder.Encode(&requestContains); err != nil {
				t.Fatal("<cmd.encode> Error in Executing Test: Contact Professor Samuels, if see this message.")
			}
		}
		<-wave1Done
		for idx := containsIdx; idx < (containsIdx + 10); idx++ {
			requestRemove := requestsRemoves[idx]
			if err := encoder.Encode(&requestRemove); err != nil {
				t.Fatal("<cmd.encode> Error in Executing Test: Contact Professor Samuels, if see this message.")
			}
		}
		<-wave2Done

		for idx := removeIdx; idx <(removeIdx + 10); idx++ {
			requestContains := requestsContains2[idx]
			if err := encoder.Encode(&requestContains); err != nil {
				t.Fatal("<cmd.encode> Error in Executing Test: Contact Professor Samuels, if see this message.")
			}
		}
		for idx := containsIdx2; idx < (containsIdx2 + 10); idx++ {
			requestRemove := requestsRemoves2[idx]
			if err := encoder.Encode(&requestRemove); err != nil {
				t.Fatal("<cmd.encode> Error in Executing Test: Contact Professor Samuels, if see this message.")
			}
		}
		<-wave3Done
		if err := encoder.Encode(&requestFeed); err != nil {
			t.Fatal("<cmd.encode> Error in Executing Test: Contact Professor Samuels, if see this message.")
		}
		<-wave4Done
		if err := encoder.Encode(&doneRequest); err != nil {
			t.Fatal("<cmd.encode> Error in Executing Test: Contact Professor Samuels, if see this message.")
		}
		inDone <- true
	}()

	go func() {
		decoder := json.NewDecoder(stdout)
		var count int
		for {
			var response _TestNormalResponse
			if count < 70 {
				if err := decoder.Decode(&response); err != nil {
					break
				}
			}
			if count >= 0 && count < 40 {
				if value, ok := responsesAdd[int(response.Id)]; ok {
					if value.Id != response.Id || value.Success != response.Success {
						t.Errorf("Add Request & Response Id and Success fields do not match. Got(%v,%v), Expected(%v,%v)",
							response.Id, response.Success, value.Id, value.Success)
					}
					count++
				} else if value, ok := responsesContains[int(response.Id)]; ok {
					if value.Id != response.Id {
						t.Errorf("Contains Request & Response Id fields do not match. Got(%v), Expected(%v)",
							response.Id, value.Id)
					}
					count++
				} else {
					t.Errorf("Received an invalid id back from twitter.go. We only added ids [0,30] but got(id):%v", response.Id)
				}
				if count == 40 {
					wave1Done <- true
				}
			}else if count >= 40 && count < 50 {
				if value, ok := responsesRemoves[int(response.Id)]; ok {
					if value.Id != response.Id || value.Success != response.Success {
						t.Errorf("Remove Request & Response Id and Success fields do not match. Got(%v,%v), Expected(%v,%v)",
							response.Id, response.Success, value.Id, value.Success)
					}
					count++
				} else {
					t.Errorf("Received an invalid id back from twitter.go. We only added ids [0,30] but got(id):%v", response.Id)
				}
				if count == 50 {
					wave2Done <- true
				}
			} else if count >= 50 && count < 70 {
				if value, ok := responsesContains2[int(response.Id)]; ok {
					if value.Id != response.Id || value.Success != response.Success {
						t.Errorf("Contains Request & Response Id and Success fields do not match. Got(%v,%v), Expected(%v,%v)",
							response.Id, response.Success, value.Id, value.Success)
					}
					count++
				} else if value, ok := responsesRemoves2[int(response.Id)]; ok {
					if value.Id != response.Id || value.Success != response.Success {
						t.Errorf("Remove Request & Response Id and Success fields do not match. Got(%v,%v), Expected(%v,%v)",
							response.Id, response.Success, value.Id, value.Success)
					}
					count++
				} else {
					t.Errorf("Received an invalid id back from twitter.go. We only added ids [0,30] but got(id):%v", response.Id)
				}
				if count == 70 {
					wave3Done <- true
				}
			} else {
				var responseFeed _TestFeedResponse
				if err := decoder.Decode(&responseFeed); err != nil {
					fmt.Printf("Got an Error\n")
					break
				}
				if responseFeed.Id != responseFeedExpected.Id  {
					t.Errorf("Feed Request & Response Id fields do not match. Got(%v), Expected(%v)",
						responseFeed.Id, responseFeedExpected.Id)
				} else {
					if len(responseFeedExpected.Feed) != len(responseFeed.Feed) {
						t.Errorf("Feed Response number of posts not equal to each other. Got(%v), Expected(%v)",
							len(responseFeed.Feed), len(responseFeedExpected.Feed))
					}
					count++
				}
				wave4Done <- true
				break
			}
		}
		if count != 71 {
			t.Errorf("Did not receieve the right amount of Add&Feed acknowledgements. Got:%v, Expected:%v", count, 71)
		}
		outDone <- true
	}()
	<-inDone
	<-outDone
	if err := cmd.Wait(); err != nil {
		t.Errorf("The automated test timed out. You may have a deadlock, starvation issue and/or you did not implement" +
			" the necessary code for passing this test.")
	}
}