package slack

import (
	"context"
	"fmt"

	"github.com/slack-go/slack"
)

// Client wraps the Slack API client
type Client struct {
	api       *slack.Client
	formatter *Formatter
}

// NewClient creates a new Slack client
func NewClient(token string) *Client {
	return &Client{
		api:       slack.New(token),
		formatter: NewFormatter(),
	}
}

// SendMessage sends a message to a Slack channel
func (c *Client) SendMessage(ctx context.Context, channelID string, msgOption slack.MsgOption) error {
	_, _, err := c.api.PostMessageContext(
		ctx,
		channelID,
		msgOption,
	)
	if err != nil {
		return fmt.Errorf("failed to send Slack message: %w", err)
	}

	return nil
}

// GetFormatter returns the message formatter
func (c *Client) GetFormatter() *Formatter {
	return c.formatter
}

// TestConnection tests the Slack API connection
func (c *Client) TestConnection(ctx context.Context) error {
	_, err := c.api.AuthTestContext(ctx)
	if err != nil {
		return fmt.Errorf("Slack API test failed: %w", err)
	}
	return nil
}
