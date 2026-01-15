// Package main provides offline integration testing using HTTP mock servers.
// Tests SNS event processing and Slack messaging without requiring live API credentials.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	scenarioFile := flag.String("scenarios", "fixtures/scenarios.json", "path to test scenarios file")
	verbose := flag.Bool("verbose", false, "enable verbose output")
	scenarioFilter := flag.String("filter", "", "run only scenarios matching this name")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Set default test environment variables
	// Tests will override these via scenario config_overrides
	loadDefaultTestEnv()

	ctx := context.Background()

	path := filepath.Join(*scenarioFile)
	raw, err := os.ReadFile(path)
	if err != nil {
		logger.Error("failed to read scenarios file", slog.Any("error", err))
		os.Exit(1)
	}

	var scenarios []TestScenario
	if err := json.Unmarshal(raw, &scenarios); err != nil {
		logger.Error("failed to parse scenarios", slog.Any("error", err))
		os.Exit(1)
	}

	passed := 0
	failed := 0
	skipped := 0

	for _, scenario := range scenarios {
		if *scenarioFilter != "" && !strings.Contains(scenario.Name, *scenarioFilter) {
			skipped++
			continue
		}

		if err := runScenario(ctx, scenario, *verbose, logger); err != nil {
			fmt.Printf("✗ FAILED: %v\n\n", err)
			failed++
		} else {
			passed++
		}
	}

	fmt.Printf("\n")
	separator := strings.Repeat("═", 60)
	fmt.Printf("%s\n", separator)
	if failed > 0 {
		fmt.Printf("  Test Results: %d passed, %d failed, %d skipped\n", passed, failed, skipped)
	} else {
		fmt.Printf("  Test Results: ✓ All %d tests passed, %d skipped\n", passed, skipped)
	}
	fmt.Printf("%s\n", separator)

	if failed > 0 {
		os.Exit(1)
	}
}

// loadDefaultTestEnv sets default environment variables for integration tests.
// Individual scenarios can override these via config_overrides.
func loadDefaultTestEnv() {
	// Set defaults only if not already set
	setEnvDefault("SLACK_TOKEN", "xoxb-test-token-12345")
	setEnvDefault("SLACK_DEFAULT_CHANNEL", "C01234DEFAULT")
	setEnvDefault("SLACK_API_URL", "https://localhost:9003/")
	setEnvDefault("SNS_ROUTES", "[]")
	setEnvDefault("FALLBACK_ENABLED", "true")
	setEnvDefault("DEBUG_ENABLED", "false")
	setEnvDefault("VALIDATE_SNS_SIGNATURES", "false")
	setEnvDefault("NOTIFY_ON_SUBSCRIPTION", "false")
}

// setEnvDefault sets an environment variable only if it's not already set.
func setEnvDefault(key, value string) {
	if os.Getenv(key) == "" {
		os.Setenv(key, value)
	}
}
