package config

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

var (
	ssmClientOnce sync.Once
	ssmClient     *ssm.Client
	ssmClientErr  error
)

// Config holds the application configuration
type Config struct {
	// General
	DebugEnabled bool

	// Slack
	SlackToken          string
	SlackDefaultChannel string

	// Routing - user provides one of these
	SNSRoutes         string // Inline JSON string
	SNSRoutesParamArn string // SSM parameter ARN (takes precedence)

	// Parsed routes (populated after loading)
	Routes []RouteConfig

	// Behavior
	FallbackEnabled      bool
	ValidateSignatures   bool
	NotifyOnSubscription bool
}

// LoadConfig loads configuration from environment variables
func LoadConfig(ctx context.Context) (*Config, error) {
	cfg := &Config{
		DebugEnabled:         getEnvBool("DEBUG_ENABLED", false),
		SlackToken:           getEnv("SLACK_TOKEN", ""),
		SlackDefaultChannel:  getEnv("SLACK_DEFAULT_CHANNEL", ""),
		SNSRoutes:            getEnv("SNS_ROUTES", ""),
		SNSRoutesParamArn:    getEnv("SNS_ROUTES_PARAM_ARN", ""),
		FallbackEnabled:      getEnvBool("FALLBACK_ENABLED", true),
		ValidateSignatures:   getEnvBool("VALIDATE_SNS_SIGNATURES", false),
		NotifyOnSubscription: getEnvBool("NOTIFY_ON_SUBSCRIPTION", false),
	}

	// Resolve SSM parameters for Slack token if needed
	if isSSMArn(cfg.SlackToken) {
		val, err := getSSMParameter(ctx, cfg.SlackToken)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch SLACK_TOKEN from SSM: %w", err)
		}
		cfg.SlackToken = val
	}

	// Load routes with precedence logic
	routesJSON, err := cfg.loadRoutes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load routes: %w", err)
	}

	// Parse routes JSON if provided
	if routesJSON != "" {
		if err := json.Unmarshal([]byte(routesJSON), &cfg.Routes); err != nil {
			return nil, fmt.Errorf("failed to parse routes JSON: %w", err)
		}

		// Compile regex patterns for all routes
		for i := range cfg.Routes {
			if err := cfg.Routes[i].CompilePatterns(); err != nil {
				return nil, fmt.Errorf("failed to compile patterns for route %q: %w",
					cfg.Routes[i].Name, err)
			}
		}
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return cfg, nil
}

// loadRoutes loads routing configuration with precedence: SSM > inline JSON
func (c *Config) loadRoutes(ctx context.Context) (string, error) {
	// Option 1: Fetch from SSM Parameter Store (takes precedence)
	if c.SNSRoutesParamArn != "" {
		if !isSSMArn(c.SNSRoutesParamArn) {
			return "", fmt.Errorf("SNS_ROUTES_PARAM_ARN is not a valid SSM ARN: %s",
				c.SNSRoutesParamArn)
		}

		routesJSON, err := getSSMParameter(ctx, c.SNSRoutesParamArn)
		if err != nil {
			return "", fmt.Errorf("failed to fetch routes from SSM parameter %s: %w",
				c.SNSRoutesParamArn, err)
		}

		return routesJSON, nil
	}

	// Option 2: Use inline JSON from environment variable
	if c.SNSRoutes != "" {
		return c.SNSRoutes, nil
	}

	// Option 3: No routes configured (will use fallback channel only)
	return "", nil
}

// Validate checks that the configuration is valid
func (c *Config) Validate() error {
	if c.SlackToken == "" {
		return fmt.Errorf("SLACK_TOKEN is required")
	}

	if c.SlackDefaultChannel == "" && c.FallbackEnabled {
		return fmt.Errorf("SLACK_DEFAULT_CHANNEL is required when FALLBACK_ENABLED=true")
	}

	if len(c.Routes) == 0 && !c.FallbackEnabled {
		return fmt.Errorf("no routes configured and fallback is disabled - no messages will be forwarded")
	}

	// Validate each route
	for i, route := range c.Routes {
		if route.Name == "" {
			return fmt.Errorf("route[%d]: name is required", i)
		}
		if route.SNSTopicArn == "" && route.SNSTopicArnPattern == "" {
			return fmt.Errorf("route[%d] (%s): must specify either sns_topic_arn or sns_topic_arn_pattern",
				i, route.Name)
		}
		if route.SlackChannelID == "" {
			return fmt.Errorf("route[%d] (%s): slack_channel_id is required", i, route.Name)
		}
	}

	return nil
}

// Redacted returns a copy of the config with sensitive values redacted
func (c *Config) Redacted() *Config {
	redacted := *c
	if redacted.SlackToken != "" {
		redacted.SlackToken = "***REDACTED***"
	}
	return &redacted
}

// NewLogger creates a structured logger based on configuration
func NewLogger(cfg *Config) *slog.Logger {
	level := slog.LevelInfo
	if cfg.DebugEnabled {
		level = slog.LevelDebug
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	// Use JSON format in Lambda environment
	if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "" {
		return slog.New(slog.NewJSONHandler(os.Stdout, opts))
	}

	// Use text format for local development
	return slog.New(slog.NewTextHandler(os.Stdout, opts))
}

// isSSMArn checks if a value is an SSM parameter ARN
func isSSMArn(value string) bool {
	return strings.HasPrefix(value, "arn:aws:ssm:")
}

// getSSMParameter fetches a parameter from SSM Parameter Store
func getSSMParameter(ctx context.Context, arn string) (string, error) {
	// Initialize SSM client lazily
	ssmClientOnce.Do(func() {
		// Parse ARN to extract region
		// Format: arn:aws:ssm:REGION:ACCOUNT:parameter/path/to/param
		parts := strings.Split(arn, ":")
		if len(parts) < 6 {
			ssmClientErr = fmt.Errorf("invalid SSM ARN format: %s", arn)
			return
		}

		region := parts[3]

		cfg, err := awsconfig.LoadDefaultConfig(ctx,
			awsconfig.WithRegion(region))
		if err != nil {
			ssmClientErr = fmt.Errorf("failed to load AWS config: %w", err)
			return
		}

		ssmClient = ssm.NewFromConfig(cfg)
	})

	if ssmClientErr != nil {
		return "", ssmClientErr
	}

	// Extract parameter name from ARN
	parts := strings.Split(arn, ":")
	if len(parts) < 6 {
		return "", fmt.Errorf("invalid SSM ARN format: %s", arn)
	}
	paramName := strings.TrimPrefix(strings.Join(parts[5:], ":"), "parameter/")

	// Fetch parameter with decryption
	input := &ssm.GetParameterInput{
		Name:           aws.String(paramName),
		WithDecryption: aws.Bool(true),
	}

	result, err := ssmClient.GetParameter(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to get SSM parameter: %w", err)
	}

	if result.Parameter == nil || result.Parameter.Value == nil {
		return "", fmt.Errorf("SSM parameter has no value")
	}

	return *result.Parameter.Value, nil
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// getEnvBool gets a boolean environment variable with a default value
func getEnvBool(key string, defaultVal bool) bool {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}

	b, err := strconv.ParseBool(val)
	if err != nil {
		return defaultVal
	}
	return b
}
