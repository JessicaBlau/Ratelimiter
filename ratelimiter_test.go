// ratelimiter_test.go
package main

import (
	"net/http"
	"net/http/httptest"
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
