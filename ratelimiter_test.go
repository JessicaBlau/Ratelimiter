// ratelimiter_test.go
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestRateLimiter(t *testing.T) {
	rateLimiter := NewRateLimiter()

	// Load the configuration from the config file
	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("Error loading config file: %v", err)
	}

	// Simulate requests for each client within a second based on the configuration
	for _, clientConfig := range config.Clients {
		clientID := clientConfig.ID
		requestMax := clientConfig.RequestMax

		for i := 0; i < requestMax; i++ {
			req := httptest.NewRequest(http.MethodGet, "/limit", nil)
			req.Header.Set(clientIDHeader, clientID)
			rec := httptest.NewRecorder()

			rateLimiter.handleLimit(rec, req)

			if rec.Code != http.StatusNoContent {
				t.Errorf("Expected HTTP status 204 for client %s, got: %d", clientID, rec.Code)
			}
		}

		// The next request for each client should be blocked
		req := httptest.NewRequest(http.MethodGet, "/limit", nil)
		req.Header.Set(clientIDHeader, clientID)
		rec := httptest.NewRecorder()

		rateLimiter.handleLimit(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected HTTP status 400 for client %s, got: %d", clientID, rec.Code)
		}
	}

	// Wait for a second to reset the request count
	time.Sleep(1 * time.Second)

	// Now, the requests for each client should be allowed again based on the configuration
	for _, clientConfig := range config.Clients {
		clientID := clientConfig.ID
		requestMax := clientConfig.RequestMax

		for i := 0; i < requestMax; i++ {
			req := httptest.NewRequest(http.MethodGet, "/limit", nil)
			req.Header.Set(clientIDHeader, clientID) // Set the clientID as a header
			rec := httptest.NewRecorder()

			rateLimiter.handleLimit(rec, req)

			if rec.Code != http.StatusNoContent {
				t.Errorf("Expected HTTP status 204 for client %s, got: %d", clientID, rec.Code)
			}
		}
	}
}

func TestRateLimiter_Concurrency(t *testing.T) {
	rateLimiter := NewRateLimiter()

	// Load the configuration from the config file
	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("Error loading config file: %v", err)
	}

	// Create a wait group to synchronize the goroutines
	var wg sync.WaitGroup

	// Simulate concurrent requests for each client within a second based on the configuration
	for _, clientConfig := range config.Clients {
		clientID := clientConfig.ID
		requestMax := clientConfig.RequestMax

		// Increase the wait group counter for each concurrent request
		wg.Add(requestMax)

		for i := 0; i < requestMax; i++ {
			go func() {
				defer wg.Done()

				req := httptest.NewRequest(http.MethodGet, "/limit", nil)
				req.Header.Set(clientIDHeader, clientID)
				rec := httptest.NewRecorder()

				rateLimiter.handleLimit(rec, req)

				if rec.Code != http.StatusNoContent {
					t.Errorf("Expected HTTP status 204 for client %s, got: %d", clientID, rec.Code)
				}
			}()
		}
	}

	// Wait for all concurrent requests to finish
	wg.Wait()

	// The next request for each client should be blocked
	for _, clientConfig := range config.Clients {
		clientID := clientConfig.ID

		req := httptest.NewRequest(http.MethodGet, "/limit", nil)
		req.Header.Set(clientIDHeader, clientID)
		rec := httptest.NewRecorder()

		rateLimiter.handleLimit(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected HTTP status 400 for client %s, got: %d", clientID, rec.Code)
		}
	}
}

func TestRateLimiter_HandleCustom(t *testing.T) {
	// Create a new rate limiter
	rl := NewRateLimiter()

	// Set up a test server
	ts := httptest.NewServer(http.HandlerFunc(rl.handleCustom))
	defer ts.Close()

	// Create a request to test
	req, err := http.NewRequest("GET", ts.URL, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	// Test case 1: Missing X-Client-ID header
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to send request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status code %d, got %d", http.StatusBadRequest, resp.StatusCode)
	}

	// Set the X-Client-ID header for the test request
	req.Header.Set(clientIDHeader, "client1")

	// Test case 2: Valid request with available tokens
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to send request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Test case 3: Exceed the token limit
	for i := 0; i < 10; i++ {
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("failed to send request: %v", err)
		}
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status code %d, got %d", http.StatusBadRequest, resp.StatusCode)
	}

	// Test case 4: Wait for the rate limiter to reset and allow more requests
	time.Sleep(requestLimitReset)

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to send request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}
}

func TestRateLimiter_HandleCustom_MultipleClients(t *testing.T) {
	// Load the configuration from the config.json file
	configJSON, err := ioutil.ReadFile("config.json")
	if err != nil {
		t.Fatalf("failed to read config.json file: %v", err)
	}

	var config Configuration
	err = json.Unmarshal(configJSON, &config)
	if err != nil {
		t.Fatalf("failed to load configuration: %v", err)
	}

	// Create a new rate limiter with the loaded configuration
	rl := NewRateLimiterWithConfig(config)

	// Set up a test server
	ts := httptest.NewServer(http.HandlerFunc(rl.handleCustom))
	defer ts.Close()

	// Define the number of concurrent requests
	numRequests := 100

	// Run the first test to use up all the allowed requests for each client
	runTest(t, ts, config, numRequests)

	// Run the second set of requests without waiting, which is expected to fail
	runTestFail(t, ts, config, numRequests)
}

func runTest(t *testing.T, ts *httptest.Server, config Configuration, numRequests int) {
	// Create a wait group to synchronize the goroutines
	var wg sync.WaitGroup
	wg.Add(numRequests)

	// Send concurrent requests from multiple clients
	for i := 0; i < numRequests; i++ {
		clientID := fmt.Sprintf("client%d", i%3+1) // client1, client2, client3

		// Create a new request to test
		req, err := http.NewRequest("GET", ts.URL, nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		req.Header.Set(clientIDHeader, clientID)

		// Start a new goroutine for each request
		go func() {
			defer wg.Done()

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("failed to send request: %v", err)
			}

			if resp.StatusCode == http.StatusOK {
				t.Logf("%s: Request allowed", clientID)
			} else {
				t.Errorf("%s: Request blocked. Status code: %d", clientID, resp.StatusCode)
			}
		}()

		// Add a delay between requests to simulate concurrent behavior
		time.Sleep(time.Second / time.Duration(config.Clients[i%3].TokensPerSec))
	}

	// Wait for all goroutines to finish
	wg.Wait()
}

func runTestFail(t *testing.T, ts *httptest.Server, config Configuration, numRequests int) {

	// Extract allowed request rate from the config
	allowedRequestsPerSecond := make(map[string]int)
	for _, clientConfig := range config.Clients {
		allowedRequestsPerSecond[clientConfig.ID] = clientConfig.RequestMax
	}

	// Create a wait group to synchronize the goroutines
	var wg sync.WaitGroup
	wg.Add(numRequests)

	// Send concurrent requests from multiple clients
	for i := 0; i < numRequests; i++ {
		clientID := fmt.Sprintf("client%d", i%3+1) // client1, client2, client3

		// Get the allowed rate for this client from the config
		allowedRate := allowedRequestsPerSecond[clientID]

		// Create a new request to test
		req, err := http.NewRequest("GET", ts.URL, nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		req.Header.Set(clientIDHeader, clientID)

		// Start a new goroutine for each request
		go func() {
			defer wg.Done()

			flag := false
			// Send multiple requests to exceed the allowed rate
			for j := 0; j < allowedRate*2; j++ {
				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					t.Fatalf("failed to send request: %v", err)
				}

				// Ensure the response body is read to reuse the connection
				// and allow subsequent requests to be processed.
				resp.Body.Close()

				// If the response status code is http.StatusBadRequest, it means the request was blocked by the rate limiter
				if resp.StatusCode == http.StatusBadRequest && j >= allowedRate {
					flag = true
					t.Logf("%s: Request blocked", clientID)
				}
			}
			// failed if it did not reach a block
			if !flag {
				t.Errorf("%s: Request was not blocked.", clientID)
			}
		}()
	}

	// Wait for all goroutines to finish
	wg.Wait()
}

// TestDockerDeployment tests the Docker container deployment
func TestDockerDeployment(t *testing.T) {
	var deployCmd *exec.Cmd

	if runtime.GOOS == "windows" {
		deployCmd = exec.Command("cmd", "/C", "deploy.bat")
	} else {
		deployCmd = exec.Command("bash", "./deploy.sh")
	}

	// Deploy the container using the appropriate script
	err := deployCmd.Run()
	if err != nil {
		t.Fatalf("Error deploying the container: %v", err)
	}
	// Wait for the container to start
	time.Sleep(2 * time.Second)

	// Stop and remove the container after the test
	stopCmd := exec.Command("docker", "stop", "ratelimiter-container")
	removeCmd := exec.Command("docker", "rm", "ratelimiter-container")
	stopCmd.Run()
	removeCmd.Run()
}
