package slack

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cruxstack/aws-sns-slack-forwarder-go/internal/sns"
	"github.com/slack-go/slack"
)

const (
	// MaxMessageLength is the max length for message content in Slack blocks
	// Slack has a 3000 char limit per block, we use 2500 to leave room for formatting
	MaxMessageLength = 2500

	// TruncationSuffix is appended to truncated messages
	TruncationSuffix = "\n\n... (message truncated, original length: %d characters)"
)

// Formatter handles formatting SNS messages for Slack
type Formatter struct{}

// NewFormatter creates a new Slack message formatter
func NewFormatter() *Formatter {
	return &Formatter{}
}

// FormatMessage converts an SNS message to a Slack message with Block Kit
func (f *Formatter) FormatMessage(msg *sns.Message) slack.MsgOption {
	// Determine color based on subject keywords
	color := f.detectSeverityColor(msg.Subject)

	// Format the message body
	formattedMessage := f.formatMessageBody(msg.Message)

	// Truncate if too long
	if len(formattedMessage) > MaxMessageLength {
		formattedMessage = f.truncateMessage(formattedMessage, MaxMessageLength)
	}

	// Build Block Kit blocks
	blocks := []slack.Block{
		// Header with topic name
		f.buildHeader(msg.TopicName),

		// Subject and timestamp
		f.buildMetadataSection(msg),

		// Message body
		f.buildMessageSection(formattedMessage),

		// Footer context
		f.buildContext(msg),
	}

	// Build message with blocks and color attachment
	return slack.MsgOptionCompose(
		slack.MsgOptionBlocks(blocks...),
		slack.MsgOptionAttachments(slack.Attachment{
			Color: color,
		}),
	)
}

// buildHeader creates the header block with topic name
func (f *Formatter) buildHeader(topicName string) *slack.HeaderBlock {
	return slack.NewHeaderBlock(
		slack.NewTextBlockObject(
			slack.PlainTextType,
			fmt.Sprintf("🔔 SNS Alert: %s", topicName),
			true, // emoji
			false,
		),
	)
}

// buildMetadataSection creates a section with subject and timestamp
func (f *Formatter) buildMetadataSection(msg *sns.Message) *slack.SectionBlock {
	fields := []*slack.TextBlockObject{}

	// Add subject if present
	if msg.Subject != "" {
		fields = append(fields, slack.NewTextBlockObject(
			slack.MarkdownType,
			fmt.Sprintf("*Subject:*\n%s", msg.Subject),
			false,
			false,
		))
	}

	// Add timestamp
	fields = append(fields, slack.NewTextBlockObject(
		slack.MarkdownType,
		fmt.Sprintf("*Timestamp:*\n%s", msg.Timestamp.UTC().Format("2006-01-02 15:04:05 UTC")),
		false,
		false,
	))

	return slack.NewSectionBlock(nil, fields, nil)
}

// buildMessageSection creates a section with the message body
func (f *Formatter) buildMessageSection(message string) *slack.SectionBlock {
	return slack.NewSectionBlock(
		slack.NewTextBlockObject(
			slack.MarkdownType,
			fmt.Sprintf("*Message:*\n```\n%s\n```", message),
			false,
			false,
		),
		nil,
		nil,
	)
}

// buildContext creates the context footer with metadata
func (f *Formatter) buildContext(msg *sns.Message) *slack.ContextBlock {
	contextText := fmt.Sprintf(
		"📍 Region: %s | Account: %s | MessageID: %s",
		msg.Region,
		msg.AccountID,
		msg.MessageID,
	)

	elements := []slack.MixedElement{
		slack.NewTextBlockObject(
			slack.MarkdownType,
			contextText,
			false,
			false,
		),
	}

	return slack.NewContextBlock("", elements...)
}

// formatMessageBody formats the message body, pretty-printing JSON if detected
func (f *Formatter) formatMessageBody(message string) string {
	// Try to parse as JSON
	var jsonData interface{}
	if err := json.Unmarshal([]byte(message), &jsonData); err == nil {
		// Valid JSON - pretty print with 2-space indent
		prettyJSON, err := json.MarshalIndent(jsonData, "", "  ")
		if err == nil {
			return string(prettyJSON)
		}
	}

	// Not JSON or failed to parse - return as-is
	return message
}

// truncateMessage truncates a message and adds a suffix indicating truncation
func (f *Formatter) truncateMessage(message string, maxLen int) string {
	// If message is within limit, return as-is
	if len(message) <= maxLen {
		return message
	}

	originalLen := len(message)
	suffix := fmt.Sprintf(TruncationSuffix, originalLen)
	truncateAt := maxLen - len(suffix)

	if truncateAt < 0 {
		truncateAt = maxLen
		suffix = "... (truncated)"
	}

	return message[:truncateAt] + suffix
}

// detectSeverityColor determines the color based on subject keywords
func (f *Formatter) detectSeverityColor(subject string) string {
	subjectLower := strings.ToLower(subject)

	// Red for errors and critical issues
	if strings.Contains(subjectLower, "error") ||
		strings.Contains(subjectLower, "critical") ||
		strings.Contains(subjectLower, "fatal") ||
		strings.Contains(subjectLower, "fail") {
		return "#d32f2f" // Red
	}

	// Yellow for warnings
	if strings.Contains(subjectLower, "warn") ||
		strings.Contains(subjectLower, "caution") ||
		strings.Contains(subjectLower, "alert") {
		return "#f57c00" // Orange/Yellow
	}

	// Green for info/success
	if strings.Contains(subjectLower, "success") ||
		strings.Contains(subjectLower, "complete") ||
		strings.Contains(subjectLower, "info") {
		return "#388e3c" // Green
	}

	// Default blue for neutral messages
	return "#1976d2" // Blue
}
