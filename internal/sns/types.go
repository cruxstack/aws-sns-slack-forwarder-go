package sns

import (
	"time"
)

// MessageType represents the type of SNS message
type MessageType string

const (
	MessageTypeNotification        MessageType = "Notification"
	MessageTypeSubscriptionConfirm MessageType = "SubscriptionConfirmation"
	MessageTypeUnsubscribeConfirm  MessageType = "UnsubscribeConfirmation"
)

// Message represents a parsed SNS message
type Message struct {
	Type      MessageType
	MessageID string
	TopicArn  string
	Subject   string
	Message   string
	Timestamp time.Time

	// Extracted metadata
	Region    string
	AccountID string
	TopicName string
}

// IsNotification returns true if this is a notification message
func (m *Message) IsNotification() bool {
	return m.Type == MessageTypeNotification
}

// IsSubscriptionConfirmation returns true if this is a subscription confirmation
func (m *Message) IsSubscriptionConfirmation() bool {
	return m.Type == MessageTypeSubscriptionConfirm
}

// IsUnsubscribeConfirmation returns true if this is an unsubscribe confirmation
func (m *Message) IsUnsubscribeConfirmation() bool {
	return m.Type == MessageTypeUnsubscribeConfirm
}
