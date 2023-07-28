// ratelimiter.go
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"
)

const (
	clientIDHeader    = "X-Client-ID" // Custom header name for clientID
	requestLimitReset = time.Second   // Request limit reset interval (1 second)
	configFile        = "config.json" // Configuration file path
)

// Client represents a single client with its request limit and request count
type Client struct {
	ID            string
	RequestMax    int
	Requests      int
	RequestMutex  sync.Mutex
	LastResetTime time.Time
}

// Configuration represents the configuration structure
type Configuration struct {
	Clients []ClientConfig `json:"clients"`
}

// ClientConfig represents the client configuration structure
type ClientConfig struct {
	ID         string `json:"ID"`
	RequestMax int    `json:"RequestMax"`
}

// RateLimiter is the main rate limiter service
type RateLimiter struct {
	clients map[string]*Client
	lock    sync.Mutex
}

// NewRateLimiter creates a new RateLimiter
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		clients: make(map[string]*Client),
	}
}

// LoadConfig loads the configuration from the configFile
func LoadConfig() (*Configuration, error) {
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	var config Configuration
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// getClient retrieves the client based on its unique ID
func (rl *RateLimiter) getClient(clientID string) *Client {
	rl.lock.Lock()
	defer rl.lock.Unlock()

	client, ok := rl.clients[clientID]
	if !ok {
		config, err := LoadConfig()
		if err != nil {
			// Set a default request limit if the config file cannot be loaded
			config = &Configuration{
				Clients: []ClientConfig{
					{ID: clientID, RequestMax: 10}, // Default request limit per second (adjust as desired)
				},
			}
		}

		var requestMax int
		for _, clientConfig := range config.Clients {
			if clientConfig.ID == clientID {
				requestMax = clientConfig.RequestMax
				break
			}
		}

		if requestMax == 0 {
			// Set a default request limit if the client is not found in the config
			requestMax = 10 // Default request limit per second (adjust as desired)
		}

		client = &Client{
			ID:            clientID,
			RequestMax:    requestMax,
			Requests:      0,
			LastResetTime: time.Now(),
		}
		rl.clients[clientID] = client
	}

	return client
}

// handleLimit handles the "/limit" endpoint
func (rl *RateLimiter) handleLimit(w http.ResponseWriter, r *http.Request) {
	clientID := r.Header.Get(clientIDHeader)
	if clientID == "" {
		http.Error(w, "X-Client-ID header missing", http.StatusBadRequest)
		return
	}

	client := rl.getClient(clientID)

	// Acquire the request mutex to ensure thread safety
	client.RequestMutex.Lock()
	defer client.RequestMutex.Unlock()

	// Reset request count if it has been 1 second since the last reset
	currentTime := time.Now()
	if currentTime.Sub(client.LastResetTime) >= requestLimitReset {
		client.Requests = 0
		client.LastResetTime = currentTime
	}

	// Check if the client has exceeded the request limit
	if client.Requests >= client.RequestMax {
		http.Error(w, "Request blocked. Too many requests.", http.StatusBadRequest)
		return
	}

	// Increment the request count
	client.Requests++

	w.WriteHeader(http.StatusNoContent)
}

func main() {
	rateLimiter := NewRateLimiter()

	http.HandleFunc("/limit", rateLimiter.handleLimit)

	fmt.Println("Rate Limiter is running on http://localhost:8080/limit")
	http.ListenAndServe(":8080", nil)
}
