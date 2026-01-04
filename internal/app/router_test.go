package app

import (
	"log/slog"
	"os"
	"testing"

	"github.com/cruxstack/aws-sns-slack-forwarder-go/internal/config"
)

func TestRouterFindChannel(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	tests := []struct {
		name            string
		routes          []config.RouteConfig
		defaultChannel  string
		fallbackEnabled bool
		topicArn        string
		subject         string
		wantChannelID   string
		wantRouteName   string
		wantMatched     bool
		wantFallback    bool
	}{
		{
			name: "exact topic ARN match",
			routes: []config.RouteConfig{
				{
					Name:           "prod-alerts",
					SNSTopicArn:    "arn:aws:sns:us-east-1:123456789012:prod-alerts",
					SlackChannelID: "C01234567890",
					Enabled:        true,
				},
			},
			defaultChannel:  "C99999999999",
			fallbackEnabled: true,
			topicArn:        "arn:aws:sns:us-east-1:123456789012:prod-alerts",
			subject:         "Test Alert",
			wantChannelID:   "C01234567890",
			wantRouteName:   "prod-alerts",
			wantMatched:     true,
			wantFallback:    false,
		},
		{
			name: "topic ARN pattern match",
			routes: []config.RouteConfig{
				{
					Name:               "staging-pattern",
					SNSTopicArnPattern: ".*:staging-.*",
					SlackChannelID:     "C11111111111",
					Enabled:            true,
				},
			},
			defaultChannel:  "C99999999999",
			fallbackEnabled: true,
			topicArn:        "arn:aws:sns:us-west-2:987654321098:staging-alerts",
			subject:         "Staging Issue",
			wantChannelID:   "C11111111111",
			wantRouteName:   "staging-pattern",
			wantMatched:     true,
			wantFallback:    false,
		},
		{
			name: "subject pattern match",
			routes: []config.RouteConfig{
				{
					Name:                  "error-alerts",
					SNSTopicArnPattern:    ".*:.*-alerts$",
					MessageSubjectPattern: "ERROR|CRITICAL",
					SlackChannelID:        "C22222222222",
					Enabled:               true,
				},
			},
			defaultChannel:  "C99999999999",
			fallbackEnabled: true,
			topicArn:        "arn:aws:sns:us-east-1:123456789012:prod-alerts",
			subject:         "ERROR: Database connection failed",
			wantChannelID:   "C22222222222",
			wantRouteName:   "error-alerts",
			wantMatched:     true,
			wantFallback:    false,
		},
		{
			name: "subject pattern no match",
			routes: []config.RouteConfig{
				{
					Name:                  "error-alerts",
					SNSTopicArnPattern:    ".*:.*-alerts$",
					MessageSubjectPattern: "ERROR|CRITICAL",
					SlackChannelID:        "C22222222222",
					Enabled:               true,
				},
			},
			defaultChannel:  "C99999999999",
			fallbackEnabled: true,
			topicArn:        "arn:aws:sns:us-east-1:123456789012:prod-alerts",
			subject:         "INFO: Normal operation",
			wantChannelID:   "C99999999999",
			wantRouteName:   "fallback",
			wantMatched:     false,
			wantFallback:    true,
		},
		{
			name: "first match wins",
			routes: []config.RouteConfig{
				{
					Name:               "first-route",
					SNSTopicArnPattern: ".*:prod-.*",
					SlackChannelID:     "C11111111111",
					Enabled:            true,
				},
				{
					Name:               "second-route",
					SNSTopicArnPattern: ".*:prod-.*",
					SlackChannelID:     "C22222222222",
					Enabled:            true,
				},
			},
			defaultChannel:  "C99999999999",
			fallbackEnabled: true,
			topicArn:        "arn:aws:sns:us-east-1:123456789012:prod-alerts",
			subject:         "Test",
			wantChannelID:   "C11111111111",
			wantRouteName:   "first-route",
			wantMatched:     true,
			wantFallback:    false,
		},
		{
			name: "disabled route skipped",
			routes: []config.RouteConfig{
				{
					Name:           "disabled-route",
					SNSTopicArn:    "arn:aws:sns:us-east-1:123456789012:prod-alerts",
					SlackChannelID: "C11111111111",
					Enabled:        false,
				},
				{
					Name:           "enabled-route",
					SNSTopicArn:    "arn:aws:sns:us-east-1:123456789012:prod-alerts",
					SlackChannelID: "C22222222222",
					Enabled:        true,
				},
			},
			defaultChannel:  "C99999999999",
			fallbackEnabled: true,
			topicArn:        "arn:aws:sns:us-east-1:123456789012:prod-alerts",
			subject:         "Test",
			wantChannelID:   "C22222222222",
			wantRouteName:   "enabled-route",
			wantMatched:     true,
			wantFallback:    false,
		},
		{
			name:            "no match with fallback enabled",
			routes:          []config.RouteConfig{},
			defaultChannel:  "C99999999999",
			fallbackEnabled: true,
			topicArn:        "arn:aws:sns:us-east-1:123456789012:unknown-topic",
			subject:         "Test",
			wantChannelID:   "C99999999999",
			wantRouteName:   "fallback",
			wantMatched:     false,
			wantFallback:    true,
		},
		{
			name: "no match with fallback disabled",
			routes: []config.RouteConfig{
				{
					Name:           "specific-topic",
					SNSTopicArn:    "arn:aws:sns:us-east-1:123456789012:specific-topic",
					SlackChannelID: "C11111111111",
					Enabled:        true,
				},
			},
			defaultChannel:  "C99999999999",
			fallbackEnabled: false,
			topicArn:        "arn:aws:sns:us-east-1:123456789012:unknown-topic",
			subject:         "Test",
			wantChannelID:   "",
			wantRouteName:   "",
			wantMatched:     false,
			wantFallback:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Compile regex patterns for routes
			for i := range tt.routes {
				if err := tt.routes[i].CompilePatterns(); err != nil {
					t.Fatalf("failed to compile route patterns: %v", err)
				}
			}

			router := NewRouter(tt.routes, tt.defaultChannel, tt.fallbackEnabled, logger)
			result := router.FindChannel(tt.topicArn, tt.subject)

			if result.ChannelID != tt.wantChannelID {
				t.Errorf("ChannelID = %v, want %v", result.ChannelID, tt.wantChannelID)
			}
			if result.RouteName != tt.wantRouteName {
				t.Errorf("RouteName = %v, want %v", result.RouteName, tt.wantRouteName)
			}
			if result.Matched != tt.wantMatched {
				t.Errorf("Matched = %v, want %v", result.Matched, tt.wantMatched)
			}
			if result.UsedFallback != tt.wantFallback {
				t.Errorf("UsedFallback = %v, want %v", result.UsedFallback, tt.wantFallback)
			}
		})
	}
}

func TestRouterComplexPatterns(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	tests := []struct {
		name          string
		routePattern  string
		testTopicArns []string
		wantMatch     []bool
	}{
		{
			name:         "ends with -errors",
			routePattern: ".*-errors$",
			testTopicArns: []string{
				"arn:aws:sns:us-east-1:123:app-errors",
				"arn:aws:sns:us-east-1:123:database-errors",
				"arn:aws:sns:us-east-1:123:app-warnings",
			},
			wantMatch: []bool{true, true, false},
		},
		{
			name:         "starts with production",
			routePattern: "^arn:aws:sns:[^:]+:[^:]+:production-.*",
			testTopicArns: []string{
				"arn:aws:sns:us-east-1:123:production-alerts",
				"arn:aws:sns:us-east-1:123:staging-alerts",
				"arn:aws:sns:us-east-1:123:production-errors",
			},
			wantMatch: []bool{true, false, true},
		},
		{
			name:         "specific region",
			routePattern: "arn:aws:sns:us-west-2:.*",
			testTopicArns: []string{
				"arn:aws:sns:us-west-2:123:any-topic",
				"arn:aws:sns:us-east-1:123:any-topic",
				"arn:aws:sns:us-west-2:456:another-topic",
			},
			wantMatch: []bool{true, false, true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			route := config.RouteConfig{
				Name:               "test-route",
				SNSTopicArnPattern: tt.routePattern,
				SlackChannelID:     "C01234567890",
				Enabled:            true,
			}

			if err := route.CompilePatterns(); err != nil {
				t.Fatalf("failed to compile pattern: %v", err)
			}

			router := NewRouter([]config.RouteConfig{route}, "C99999", true, logger)

			for i, topicArn := range tt.testTopicArns {
				result := router.FindChannel(topicArn, "")
				gotMatch := result.Matched

				if gotMatch != tt.wantMatch[i] {
					t.Errorf("topic %q: matched = %v, want %v", topicArn, gotMatch, tt.wantMatch[i])
				}
			}
		})
	}
}
