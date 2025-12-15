package app

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/aws/aws-lambda-go/events"
	"github.com/cruxstack/aws-sns-slackbot/internal/config"
	"github.com/cruxstack/aws-sns-slackbot/internal/slack"
	"github.com/cruxstack/aws-sns-slackbot/internal/sns"
)

// App is the main application
type App struct {
	config *config.Config
	logger *slog.Logger
	router *Router
	slack  *slack.Client
}

// New creates a new App instance
func New(ctx context.Context) (*App, error) {
	// Load configuration
	cfg, err := config.LoadConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Create logger
	logger := config.NewLogger(cfg)

	logger.Info("initializing application",
		"debug_enabled", cfg.DebugEnabled,
		"routes_count", len(cfg.Routes),
		"fallback_enabled", cfg.FallbackEnabled)

	// Create Slack client
	slackClient := slack.NewClient(cfg.SlackToken)

	// Test Slack connection
	if err := slackClient.TestConnection(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to Slack: %w", err)
	}
	logger.Info("slack connection established")

	// Create router
	router := NewRouter(
		cfg.Routes,
		cfg.SlackDefaultChannel,
		cfg.FallbackEnabled,
		logger,
	)

	return &App{
		config: cfg,
		logger: logger,
		router: router,
		slack:  slackClient,
	}, nil
}

// HandleSNSEvent processes an SNS Lambda event
func (a *App) HandleSNSEvent(ctx context.Context, event events.SNSEvent) error {
	a.logger.Info("processing SNS event", "records", len(event.Records))

	// Parse SNS messages
	messages, err := sns.ParseEvent(event)
	if err != nil {
		return fmt.Errorf("failed to parse SNS event: %w", err)
	}

	// Process each message
	for _, msg := range messages {
		if err := a.processMessage(ctx, msg); err != nil {
			// Log error but continue processing other messages
			a.logger.Error("failed to process message",
				"message_id", msg.MessageID,
				"topic_arn", msg.TopicArn,
				"error", err)
		}
	}

	return nil
}

// processMessage processes a single SNS message
func (a *App) processMessage(ctx context.Context, msg *sns.Message) error {
	a.logger.Debug("processing message",
		"message_id", msg.MessageID,
		"type", msg.Type,
		"topic_arn", msg.TopicArn,
		"subject", msg.Subject)

	// Handle subscription confirmation
	if msg.IsSubscriptionConfirmation() {
		return a.handleSubscriptionConfirmation(ctx, msg)
	}

	// Handle regular notification
	if msg.IsNotification() {
		return a.handleNotification(ctx, msg)
	}

	// Unknown message type
	a.logger.Warn("unknown message type, skipping",
		"type", msg.Type,
		"message_id", msg.MessageID)

	return nil
}

// handleNotification processes a notification message
func (a *App) handleNotification(ctx context.Context, msg *sns.Message) error {
	// Find the appropriate channel
	result := a.router.FindChannel(msg.TopicArn, msg.Subject)

	if !result.Matched && !result.UsedFallback {
		// No route matched and no fallback - skip message
		a.logger.Warn("message skipped - no matching route",
			"message_id", msg.MessageID,
			"topic_arn", msg.TopicArn)
		return nil
	}

	// Format the message
	formatter := a.slack.GetFormatter()
	msgOption := formatter.FormatMessage(msg)

	// Send to Slack
	if err := a.slack.SendMessage(ctx, result.ChannelID, msgOption); err != nil {
		return fmt.Errorf("failed to send message to Slack channel %s: %w",
			result.ChannelID, err)
	}

	a.logger.Info("message sent to Slack",
		"message_id", msg.MessageID,
		"channel", result.ChannelID,
		"route", result.RouteName,
		"fallback", result.UsedFallback)

	return nil
}

// handleSubscriptionConfirmation processes a subscription confirmation message
func (a *App) handleSubscriptionConfirmation(ctx context.Context, msg *sns.Message) error {
	a.logger.Info("received subscription confirmation",
		"topic_arn", msg.TopicArn,
		"message_id", msg.MessageID)

	// Note: SNS subscription confirmations in Lambda are automatically handled
	// by AWS when the Lambda is configured as an SNS trigger. No action needed.

	// Optionally notify about the subscription
	if a.config.NotifyOnSubscription && a.config.SlackDefaultChannel != "" {
		// Send a notification to the default channel
		a.logger.Info("sending subscription notification to Slack")
		// This could be implemented if needed
	}

	return nil
}

// GetConfig returns the application configuration (redacted)
func (a *App) GetConfig() *config.Config {
	return a.config.Redacted()
}

// GetLogger returns the application logger
func (a *App) GetLogger() *slog.Logger {
	return a.logger
}
