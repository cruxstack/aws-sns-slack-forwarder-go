package slack

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/slack-go/slack"
)

// Client wraps the Slack API client
type Client struct {
	api       *slack.Client
	formatter *Formatter
}

// NewClient creates a new Slack client
func NewClient(token string) *Client {
	// Support override url for integration testing only
	opts := []slack.Option{}
	if apiURL := os.Getenv("SLACK_API_URL"); apiURL != "" {
		opts = append(opts, slack.OptionAPIURL(apiURL))

		// Use custom HTTP transport if configured (for integration tests with self-signed certs)
		// Tests configure http.DefaultTransport with custom TLS config
		if httpClient, ok := getTestHTTPClient(); ok {
			opts = append(opts, slack.OptionHTTPClient(httpClient))
		}
	}

	return &Client{
		api:       slack.New(token, opts...),
		formatter: NewFormatter(),
	}
}

// getTestHTTPClient returns a custom HTTP client for testing if configured
func getTestHTTPClient() (*http.Client, bool) {
	// Check if http.DefaultTransport is customized (not the default)
	// Integration tests set this to trust self-signed certificates
	transport, ok := http.DefaultTransport.(*http.Transport)
	if !ok || transport == nil {
		return nil, false
	}

	// If TLS config is customized, assume this is for testing
	if transport.TLSClientConfig != nil && transport.TLSClientConfig.RootCAs != nil {
		return &http.Client{Transport: transport}, true
	}

	return nil, false
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
