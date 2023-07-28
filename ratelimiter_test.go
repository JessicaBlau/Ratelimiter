// ratelimiter_test.go
package main

import (
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
	rateLimiter := NewRateLimiter()

	// Test with client1
	clientID := "client1"

	// Send 5 requests (within limit)
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/custom", nil)
		req.Header.Set(clientIDHeader, clientID)
		rec := httptest.NewRecorder()

		rateLimiter.handleCustom(rec, req)

		if rec.Code != http.StatusOK {
			client := rateLimiter.getClient(clientID)
			t.Log(client.RateLimiter.Available(), client.Requests)
			t.Errorf("Expected HTTP status 200, got: %d", rec.Code)
		}
	}

	// Send 6th request (exceeding the limit)
	req := httptest.NewRequest(http.MethodGet, "/custom", nil)
	req.Header.Set(clientIDHeader, clientID)
	rec := httptest.NewRecorder()

	rateLimiter.handleCustom(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected HTTP status 400, got: %d", rec.Code)
	}

}

func TestRateLimiter_HandleCustom_MultipleClients(t *testing.T) {
	rateLimiter := NewRateLimiter()

	// Test with client1
	client1ID := "client1"

	// Send 5 requests for client1 (within limit)
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/custom", nil)
		req.Header.Set(clientIDHeader, client1ID)
		rec := httptest.NewRecorder()

		rateLimiter.handleCustom(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected HTTP status 200, got: %d", rec.Code)
		}
	}

	// Test with client2
	client2ID := "client2"

	// Send 10 requests for client2 (within limit) but not enough tokens
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/custom", nil)
		req.Header.Set(clientIDHeader, client2ID)
		rec := httptest.NewRecorder()

		rateLimiter.handleCustom(rec, req)

		if rec.Code != http.StatusOK {
			body, _ := ioutil.ReadAll(rec.Body)
			errorMessage := string(body)
			// to do: change to a binary variable and not a string
			if errorMessage != "Request blocked. No more tokens.\n" {
				t.Errorf("Expected HTTP status 200, got: %d", rec.Code)
			}
		}
	}
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
