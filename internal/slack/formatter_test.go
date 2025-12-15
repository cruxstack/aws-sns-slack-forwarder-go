package slack

import (
	"strings"
	"testing"
	"time"

	"github.com/cruxstack/aws-sns-slackbot/internal/sns"
)

func TestDetectSeverityColor(t *testing.T) {
	tests := []struct {
		name    string
		subject string
		want    string
	}{
		{
			name:    "error keyword",
			subject: "ERROR: Database connection failed",
			want:    "#d32f2f", // Red
		},
		{
			name:    "critical keyword",
			subject: "CRITICAL: System failure",
			want:    "#d32f2f", // Red
		},
		{
			name:    "fatal keyword",
			subject: "FATAL: Application crashed",
			want:    "#d32f2f", // Red
		},
		{
			name:    "fail keyword",
			subject: "FAILED: Deployment failed",
			want:    "#d32f2f", // Red
		},
		{
			name:    "warning keyword",
			subject: "WARNING: High memory usage",
			want:    "#f57c00", // Orange
		},
		{
			name:    "caution keyword",
			subject: "CAUTION: Approaching limit",
			want:    "#f57c00", // Orange
		},
		{
			name:    "alert keyword",
			subject: "ALERT: Unusual activity",
			want:    "#f57c00", // Orange
		},
		{
			name:    "success keyword",
			subject: "SUCCESS: Deployment completed",
			want:    "#388e3c", // Green
		},
		{
			name:    "complete keyword",
			subject: "COMPLETE: Backup finished",
			want:    "#388e3c", // Green
		},
		{
			name:    "info keyword",
			subject: "INFO: Status update",
			want:    "#388e3c", // Green
		},
		{
			name:    "neutral message",
			subject: "System notification",
			want:    "#1976d2", // Blue (default)
		},
		{
			name:    "empty subject",
			subject: "",
			want:    "#1976d2", // Blue (default)
		},
	}

	f := NewFormatter()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := f.detectSeverityColor(tt.subject)
			if got != tt.want {
				t.Errorf("detectSeverityColor(%q) = %q, want %q", tt.subject, got, tt.want)
			}
		})
	}
}

func TestFormatMessageBody(t *testing.T) {
	tests := []struct {
		name       string
		message    string
		wantJSON   bool
		wantIndent bool
	}{
		{
			name:       "valid JSON object",
			message:    `{"service":"api","status":"error","code":500}`,
			wantJSON:   true,
			wantIndent: true,
		},
		{
			name:       "valid JSON array",
			message:    `["item1","item2","item3"]`,
			wantJSON:   true,
			wantIndent: true,
		},
		{
			name:       "plain text",
			message:    "This is a plain text message",
			wantJSON:   false,
			wantIndent: false,
		},
		{
			name:       "invalid JSON",
			message:    `{"incomplete": `,
			wantJSON:   false,
			wantIndent: false,
		},
		{
			name:       "empty message",
			message:    "",
			wantJSON:   false,
			wantIndent: false,
		},
	}

	f := NewFormatter()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := f.formatMessageBody(tt.message)

			if tt.wantJSON {
				// Should be pretty-printed JSON with 2-space indent
				if !strings.Contains(got, "\n") {
					t.Error("expected pretty-printed JSON with newlines")
				}
				if !strings.Contains(got, "  ") {
					t.Error("expected 2-space indentation")
				}
			} else {
				// Should return original message
				if got != tt.message {
					t.Errorf("formatMessageBody() = %q, want %q", got, tt.message)
				}
			}
		})
	}
}

func TestTruncateMessage(t *testing.T) {
	tests := []struct {
		name       string
		message    string
		maxLen     int
		wantLen    int
		wantSuffix bool
	}{
		{
			name:       "short message not truncated",
			message:    "Short message",
			maxLen:     100,
			wantLen:    13,
			wantSuffix: false,
		},
		{
			name:       "long message truncated",
			message:    strings.Repeat("a", 1000),
			maxLen:     100,
			wantLen:    100,
			wantSuffix: true,
		},
		{
			name:       "exact length",
			message:    strings.Repeat("b", 100),
			maxLen:     100,
			wantLen:    100,
			wantSuffix: false,
		},
		{
			name:       "one over max",
			message:    strings.Repeat("c", 101),
			maxLen:     100,
			wantLen:    100,
			wantSuffix: true,
		},
	}

	f := NewFormatter()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := f.truncateMessage(tt.message, tt.maxLen)

			if len(got) != tt.wantLen {
				t.Errorf("truncateMessage() length = %d, want %d", len(got), tt.wantLen)
			}

			if tt.wantSuffix {
				if !strings.Contains(got, "truncated") {
					t.Error("expected truncation notice in message")
				}
				if !strings.Contains(got, "original length:") {
					t.Error("expected original length indicator")
				}
			}
		})
	}
}

func TestFormatMessage(t *testing.T) {
	tests := []struct {
		name string
		msg  *sns.Message
		// We can't easily test the full Block Kit structure,
		// but we can test that the function doesn't panic
	}{
		{
			name: "standard notification",
			msg: &sns.Message{
				Type:      sns.MessageTypeNotification,
				MessageID: "msg-123",
				TopicArn:  "arn:aws:sns:us-east-1:123456789012:prod-alerts",
				Subject:   "ERROR: Database failure",
				Message:   `{"error":"connection timeout","service":"database"}`,
				Timestamp: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
				Region:    "us-east-1",
				AccountID: "123456789012",
				TopicName: "prod-alerts",
			},
		},
		{
			name: "message without subject",
			msg: &sns.Message{
				Type:      sns.MessageTypeNotification,
				MessageID: "msg-456",
				TopicArn:  "arn:aws:sns:us-west-2:987654321098:staging-topic",
				Subject:   "",
				Message:   "Plain text message",
				Timestamp: time.Now(),
				Region:    "us-west-2",
				AccountID: "987654321098",
				TopicName: "staging-topic",
			},
		},
		{
			name: "very long message",
			msg: &sns.Message{
				Type:      sns.MessageTypeNotification,
				MessageID: "msg-789",
				TopicArn:  "arn:aws:sns:us-east-1:123:test",
				Subject:   "Long message",
				Message:   strings.Repeat("This is a very long message. ", 200),
				Timestamp: time.Now(),
				Region:    "us-east-1",
				AccountID: "123",
				TopicName: "test",
			},
		},
	}

	f := NewFormatter()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that FormatMessage doesn't panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("FormatMessage() panicked: %v", r)
				}
			}()

			msgOption := f.FormatMessage(tt.msg)
			if msgOption == nil {
				t.Error("FormatMessage() returned nil")
			}
		})
	}
}

func TestMaxMessageLength(t *testing.T) {
	// Test that MaxMessageLength constant is reasonable
	if MaxMessageLength < 1000 {
		t.Error("MaxMessageLength seems too small")
	}
	if MaxMessageLength > 3000 {
		t.Error("MaxMessageLength exceeds Slack block limit")
	}
}
