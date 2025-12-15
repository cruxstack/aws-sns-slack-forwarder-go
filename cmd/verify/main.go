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

	envPath := filepath.Join("cmd", "verify", ".env.test")
	if _, err := os.Stat(envPath); err == nil {
		fmt.Printf("Using %s for test environment\n", envPath)
		// Note: We don't use godotenv to avoid the dependency
		// Tests will set required env vars programmatically
	}

	ctx := context.Background()

	path := filepath.Join(*scenarioFile)
	raw, err := os.ReadFile(path)
	if err != nil {
		logger.Error("failed to read scenarios file", slog.String("error", err.Error()))
		os.Exit(1)
	}

	var scenarios []TestScenario
	if err := json.Unmarshal(raw, &scenarios); err != nil {
		logger.Error("failed to parse scenarios", slog.String("error", err.Error()))
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
