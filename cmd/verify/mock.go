package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// RequestRecord captures details of an HTTP request made during testing.
type RequestRecord struct {
	Timestamp time.Time           `json:"timestamp"`
	Method    string              `json:"method"`
	Host      string              `json:"host"`
	Path      string              `json:"path"`
	Query     string              `json:"query,omitempty"`
	Headers   map[string][]string `json:"headers"`
	Body      string              `json:"body,omitempty"`
}

// MockResponse defines a canned HTTP response returned by the mock server.
type MockResponse struct {
	Service    string            `json:"service"`
	Method     string            `json:"method"`
	Path       string            `json:"path"`
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers,omitempty"`
	Body       string            `json:"body"`
}

// MockServer simulates an HTTP API service for integration testing.
type MockServer struct {
	name      string
	mu        sync.Mutex
	requests  []RequestRecord
	responses map[string]MockResponse
	verbose   bool
}

// NewMockServer creates a new mock HTTP server with canned responses.
func NewMockServer(name string, responses []MockResponse, verbose bool) *MockServer {
	respMap := make(map[string]MockResponse)
	for _, r := range responses {
		key := fmt.Sprintf("%s:%s", r.Method, r.Path)
		respMap[key] = r
	}
	return &MockServer{
		name:      name,
		requests:  make([]RequestRecord, 0),
		responses: respMap,
		verbose:   verbose,
	}
}

// ServeHTTP records the request and returns a matching mock response.
func (ms *MockServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	r.Body.Close()

	rec := RequestRecord{
		Timestamp: time.Now(),
		Method:    r.Method,
		Host:      r.Host,
		Path:      r.URL.Path,
		Query:     r.URL.RawQuery,
		Headers:   r.Header,
		Body:      string(body),
	}

	ms.mu.Lock()
	ms.requests = append(ms.requests, rec)
	ms.mu.Unlock()

	if ms.verbose {
		serviceName := fmt.Sprintf("%-6s", ms.name)
		fmt.Printf("  → %s %-4s %s\n", serviceName, r.Method, r.URL.Path)
	}

	// Try exact match first
	key := fmt.Sprintf("%s:%s", r.Method, r.URL.Path)
	if resp, ok := ms.responses[key]; ok {
		ms.writeResponse(w, resp)
		return
	}

	// Try wildcard pattern matching
	for key, resp := range ms.responses {
		parts := strings.Split(key, ":")
		if len(parts) == 2 {
			method, pattern := parts[0], parts[1]
			if method == r.Method && matchPath(r.URL.Path, pattern) {
				ms.writeResponse(w, resp)
				return
			}
		}
	}

	// No match found
	if ms.verbose {
		serviceName := fmt.Sprintf("%-6s", ms.name)
		fmt.Printf("  ✗ %s No mock response for: %s %s\n", serviceName, r.Method, r.URL.Path)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(`{"message":"not found in mock"}`))
}

// writeResponse writes a mock response to the http.ResponseWriter
func (ms *MockServer) writeResponse(w http.ResponseWriter, resp MockResponse) {
	for k, v := range resp.Headers {
		w.Header().Set(k, v)
	}
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "application/json")
	}
	w.WriteHeader(resp.StatusCode)
	w.Write([]byte(resp.Body))
}

// GetRequests returns all HTTP requests captured by the mock server.
func (ms *MockServer) GetRequests() []RequestRecord {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	reqs := make([]RequestRecord, len(ms.requests))
	copy(reqs, ms.requests)
	return reqs
}

// Reset clears all recorded requests from the mock server.
func (ms *MockServer) Reset() {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.requests = make([]RequestRecord, 0)
}

// matchPath checks if an actual HTTP path matches an expected pattern.
// Supports wildcards (*) in the expected pattern.
func matchPath(actual, expected string) bool {
	if expected == "*" {
		return true
	}

	if actual == expected {
		return true
	}

	if strings.Contains(expected, "*") {
		parts := strings.Split(expected, "*")

		if len(parts) == 2 {
			return strings.HasPrefix(actual, parts[0]) && strings.HasSuffix(actual, parts[1])
		}

		pos := 0
		for i, part := range parts {
			if part == "" {
				continue
			}

			idx := strings.Index(actual[pos:], part)
			if idx == -1 {
				return false
			}

			if i == 0 && idx != 0 {
				return false
			}

			pos += idx + len(part)
		}

		if parts[len(parts)-1] != "" {
			return strings.HasSuffix(actual, parts[len(parts)-1])
		}

		return true
	}

	return false
}
