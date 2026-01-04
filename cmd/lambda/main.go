package main

import (
	"context"
	"log/slog"
	"sync"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/cruxstack/aws-sns-slack-forwarder-go/internal/app"
)

var (
	// App instance - initialized once per container lifecycle
	initOnce sync.Once
	appInst  *app.App
	logger   *slog.Logger
	initErr  error
)

func main() {
	lambda.Start(handler)
}

// handler is the main Lambda handler function
func handler(ctx context.Context, event events.SNSEvent) error {
	// Initialize app once per container lifecycle
	initOnce.Do(func() {
		logger = slog.Default()
		logger.Info("initializing Lambda function")

		appInst, initErr = app.New(ctx)
		if initErr != nil {
			logger.Error("failed to initialize app", "error", initErr)
			return
		}

		// Update logger to use app's configured logger
		logger = appInst.GetLogger()
		logger.Info("Lambda function initialized successfully")
	})

	// Return initialization error if it occurred
	if initErr != nil {
		return initErr
	}

	// Process the SNS event
	return appInst.HandleSNSEvent(ctx, event)
}
