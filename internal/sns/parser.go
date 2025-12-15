package sns

import (
	"fmt"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

// ParseEvent parses an SNS Lambda event into Message structs
func ParseEvent(event events.SNSEvent) ([]*Message, error) {
	messages := make([]*Message, 0, len(event.Records))

	for _, record := range event.Records {
		msg, err := parseRecord(record)
		if err != nil {
			return nil, fmt.Errorf("failed to parse SNS record: %w", err)
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// parseRecord converts an SNS event record to a Message
func parseRecord(record events.SNSEventRecord) (*Message, error) {
	snsRecord := record.SNS

	msg := &Message{
		Type:      MessageType(snsRecord.Type),
		MessageID: snsRecord.MessageID,
		TopicArn:  snsRecord.TopicArn,
		Subject:   snsRecord.Subject,
		Message:   snsRecord.Message,
		Timestamp: snsRecord.Timestamp,
	}

	// Extract metadata from topic ARN
	// Format: arn:aws:sns:REGION:ACCOUNT_ID:TOPIC_NAME
	msg.extractMetadata()

	return msg, nil
}

// extractMetadata extracts region, account ID, and topic name from topic ARN
func (m *Message) extractMetadata() {
	parts := strings.Split(m.TopicArn, ":")
	if len(parts) >= 6 {
		m.Region = parts[3]
		m.AccountID = parts[4]
		m.TopicName = parts[5]
	}
}
