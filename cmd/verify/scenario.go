package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/cruxstack/aws-sns-slack-forwarder-go/internal/app"
)

// TestScenario defines a test case with input SNS events and expected outcomes.
type TestScenario struct {
	Name            string            `json:"name"`
	Description     string            `json:"description,omitempty"`
	SNSPayload      json.RawMessage   `json:"sns_payload"`
	ConfigOverrides map[string]string `json:"config_overrides,omitempty"`
	ExpectedCalls   []ExpectedCall    `json:"expected_calls"`
	MockResponses   []MockResponse    `json:"mock_responses"`
	ExpectError     bool              `json:"expect_error,omitempty"`
}

// ExpectedCall defines an HTTP API call the test expects the application to make.
type ExpectedCall struct {
	Service string `json:"service"`
	Method  string `json:"method"`
	Path    string `json:"path"`
}

// runScenario executes a single test scenario with mock HTTP servers.
func runScenario(ctx context.Context, scenario TestScenario, verbose bool, logger *slog.Logger) error {
	startTime := time.Now()

	fmt.Printf("\n▶ Running: %s\n", scenario.Name)
	if scenario.Description != "" {
		fmt.Printf("  %s\n", scenario.Description)
	}

	// Separate mock responses by service
	slackResponses := []MockResponse{}
	for _, resp := range scenario.MockResponses {
		if resp.Service == "slack" {
			slackResponses = append(slackResponses, resp)
		}
	}

	slackMock := NewMockServer("Slack", slackResponses, verbose)

	// Generate TLS cert for HTTPS mock server
	tlsCert, certPool, err := generateSelfSignedCert()
	if err != nil {
		return fmt.Errorf("generate cert: %w", err)
	}

	// Start Slack mock server
	slackServer := &http.Server{
		Addr:    "localhost:9003",
		Handler: slackMock,
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{tlsCert},
		},
	}

	slackReady := make(chan bool)
	go func() {
		slackReady <- true
		if err := slackServer.ListenAndServeTLS("", ""); err != http.ErrServerClosed {
			logger.Error("slack mock server error", slog.String("error", err.Error()))
		}
	}()

	<-slackReady
	time.Sleep(100 * time.Millisecond)

	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		slackServer.Shutdown(shutdownCtx)
	}()

	// Configure HTTP client to trust self-signed cert
	http.DefaultTransport = &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs: certPool,
		},
	}

	// Apply scenario-specific config overrides
	// Default test env vars are already set by main.go's loadDefaultTestEnv()
	for key, value := range scenario.ConfigOverrides {
		os.Setenv(key, value)
	}

	// Parse SNS event
	var snsEvent events.SNSEvent
	if err := json.Unmarshal(scenario.SNSPayload, &snsEvent); err != nil {
		return fmt.Errorf("parse SNS payload: %w", err)
	}

	if verbose {
		fmt.Printf("\n  Application Output:\n")
	}

	// Initialize app with test configuration
	// The app will load config from environment variables (set above and from .env.test)
	a, err := app.New(ctx)
	if err != nil {
		return fmt.Errorf("app initialization failed: %w", err)
	}

	if verbose {
		fmt.Printf("  ✓ App initialized successfully\n")
	}

	// Process the SNS event
	processErr := a.HandleSNSEvent(ctx, snsEvent)

	if scenario.ExpectError {
		if processErr == nil {
			return fmt.Errorf("expected error but processing succeeded")
		}
		if verbose {
			fmt.Printf("  ✓ Expected error occurred: %v\n", processErr)
		}
	} else {
		if processErr != nil {
			return fmt.Errorf("process event failed: %w", processErr)
		}
		if verbose {
			fmt.Printf("  ✓ SNS event processed successfully\n")
		}
	}

	// Allow time for async operations to complete
	time.Sleep(500 * time.Millisecond)

	// Validate expected API calls
	slackReqs := slackMock.GetRequests()

	allReqs := make(map[string][]RequestRecord)
	allReqs["slack"] = slackReqs

	totalCalls := len(slackReqs)

	if verbose {
		fmt.Printf("\n")
	}

	if err := validateExpectedCalls(scenario.ExpectedCalls, allReqs); err != nil {
		fmt.Printf("\n  Validation:\n")
		fmt.Printf("  ✗ FAILED: %v\n", err)
		fmt.Printf("\n  All captured requests:\n")
		if len(slackReqs) > 0 {
			fmt.Printf("    Slack (%d):\n", len(slackReqs))
			for i, req := range slackReqs {
				fmt.Printf("      [%d] %s %s\n", i+1, req.Method, req.Path)
			}
		}
		return err
	}

	duration := time.Since(startTime)

	if verbose {
		fmt.Printf("  Validation:\n")
		fmt.Printf("  ✓ All expected calls verified (%d total)\n", totalCalls)
		fmt.Printf("\n")
	}

	fmt.Printf("✓ PASSED (Duration: %.2fs)\n", duration.Seconds())
	return nil
}

// validateExpectedCalls verifies that all expected HTTP calls were captured.
func validateExpectedCalls(expected []ExpectedCall, allReqs map[string][]RequestRecord) error {
	for _, exp := range expected {
		reqs := allReqs[exp.Service]
		found := false
		for _, req := range reqs {
			if req.Method == exp.Method && matchPath(req.Path, exp.Path) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("expected call not found: %s %s %s", exp.Service, exp.Method, exp.Path)
		}
	}
	return nil
}
