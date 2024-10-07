package main

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"os"
	"io/ioutil"
	"time"
	"bytes"
	"strings"
	"sync"

	"github.com/joho/godotenv"
)

func testRequest(ctx context.Context, t *testing.T, port string, requestNum int) {
	fmt.Printf("Sending request %d\n", requestNum)
	reqBody := bytes.NewBuffer([]byte(`{"somejson":"value"}`))
	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("http://localhost:%s?first=value", port), reqBody)
	if err != nil {
		t.Fatalf("Failed to create POST request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to send GET request: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	if !strings.Contains(string(body), "POST") { 
		t.Fatalf("Did not echo METHOD, got %v", string(body))
	}
	if !strings.Contains(string(body), "somejson") { 
		t.Fatalf("Did not echo BODY, got %v", string(body))
	}
	if !strings.Contains(string(body), "first") { 
		t.Fatalf("Did not echo QUERY, got %v", string(body))
	}
}

func TestConcurrently(t *testing.T) {
	// Start server
	go main()

	err := godotenv.Load()
	if err != nil {
		t.Fatalf("Error loading .env file: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()


	port := os.Getenv("PORT")
	fmt.Printf("Running test on port %s\n", port)

	// run testRequest concurrently 10 times, and validate responses in a channel slice
	numRequests := 10
	results := make(chan error, numRequests)
	var wg sync.WaitGroup

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					results <- fmt.Errorf("panic occurred: %v", r)
				}
			}()
			testRequest(ctx, t, port, i)
			results <- nil
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	for i := 0; i < numRequests; i++ {
		if err := <-results; err != nil {
			t.Errorf("Request %d failed: %v", i, err)
		}
	}
}