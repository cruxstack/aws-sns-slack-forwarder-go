package sns

import (
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
)

func TestParseEvent(t *testing.T) {
	tests := []struct {
		name      string
		event     events.SNSEvent
		wantCount int
		wantErr   bool
	}{
		{
			name: "single notification message",
			event: events.SNSEvent{
				Records: []events.SNSEventRecord{
					{
						EventSource: "aws:sns",
						SNS: events.SNSEntity{
							Type:      "Notification",
							MessageID: "msg-123",
							TopicArn:  "arn:aws:sns:us-east-1:123456789012:test-topic",
							Subject:   "Test Alert",
							Message:   "This is a test message",
							Timestamp: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
						},
					},
				},
			},
			wantCount: 1,
			wantErr:   false,
		},
		{
			name: "multiple messages",
			event: events.SNSEvent{
				Records: []events.SNSEventRecord{
					{
						EventSource: "aws:sns",
						SNS: events.SNSEntity{
							Type:      "Notification",
							MessageID: "msg-1",
							TopicArn:  "arn:aws:sns:us-east-1:123456789012:topic-1",
							Subject:   "Alert 1",
							Message:   "Message 1",
							Timestamp: time.Now(),
						},
					},
					{
						EventSource: "aws:sns",
						SNS: events.SNSEntity{
							Type:      "Notification",
							MessageID: "msg-2",
							TopicArn:  "arn:aws:sns:us-east-1:123456789012:topic-2",
							Subject:   "Alert 2",
							Message:   "Message 2",
							Timestamp: time.Now(),
						},
					},
				},
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name: "empty event",
			event: events.SNSEvent{
				Records: []events.SNSEventRecord{},
			},
			wantCount: 0,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseEvent(tt.event)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseEvent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != tt.wantCount {
				t.Errorf("ParseEvent() got %d messages, want %d", len(got), tt.wantCount)
			}
		})
	}
}

func TestParseRecord(t *testing.T) {
	tests := []struct {
		name    string
		record  events.SNSEventRecord
		want    *Message
		wantErr bool
	}{
		{
			name: "standard notification",
			record: events.SNSEventRecord{
				EventSource: "aws:sns",
				SNS: events.SNSEntity{
					Type:      "Notification",
					MessageID: "95df01b4-ee98-5cb9-9903-4c221d41eb5e",
					TopicArn:  "arn:aws:sns:us-east-1:123456789012:prod-alerts",
					Subject:   "ERROR: Application failure",
					Message:   "Service is down",
					Timestamp: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
				},
			},
			want: &Message{
				Type:      MessageTypeNotification,
				MessageID: "95df01b4-ee98-5cb9-9903-4c221d41eb5e",
				TopicArn:  "arn:aws:sns:us-east-1:123456789012:prod-alerts",
				Subject:   "ERROR: Application failure",
				Message:   "Service is down",
				Timestamp: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
				Region:    "us-east-1",
				AccountID: "123456789012",
				TopicName: "prod-alerts",
			},
			wantErr: false,
		},
		{
			name: "subscription confirmation",
			record: events.SNSEventRecord{
				EventSource: "aws:sns",
				SNS: events.SNSEntity{
					Type:      "SubscriptionConfirmation",
					MessageID: "sub-conf-123",
					TopicArn:  "arn:aws:sns:us-west-2:987654321098:staging-topic",
					Subject:   "",
					Message:   "You have chosen to subscribe...",
					Timestamp: time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC),
				},
			},
			want: &Message{
				Type:      MessageTypeSubscriptionConfirm,
				MessageID: "sub-conf-123",
				TopicArn:  "arn:aws:sns:us-west-2:987654321098:staging-topic",
				Subject:   "",
				Message:   "You have chosen to subscribe...",
				Timestamp: time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC),
				Region:    "us-west-2",
				AccountID: "987654321098",
				TopicName: "staging-topic",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRecord(tt.record)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseRecord() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got == nil && tt.want != nil {
				t.Error("parseRecord() returned nil, expected message")
				return
			}
			if got != nil && tt.want != nil {
				if got.Type != tt.want.Type {
					t.Errorf("Type = %v, want %v", got.Type, tt.want.Type)
				}
				if got.MessageID != tt.want.MessageID {
					t.Errorf("MessageID = %v, want %v", got.MessageID, tt.want.MessageID)
				}
				if got.TopicArn != tt.want.TopicArn {
					t.Errorf("TopicArn = %v, want %v", got.TopicArn, tt.want.TopicArn)
				}
				if got.Region != tt.want.Region {
					t.Errorf("Region = %v, want %v", got.Region, tt.want.Region)
				}
				if got.AccountID != tt.want.AccountID {
					t.Errorf("AccountID = %v, want %v", got.AccountID, tt.want.AccountID)
				}
				if got.TopicName != tt.want.TopicName {
					t.Errorf("TopicName = %v, want %v", got.TopicName, tt.want.TopicName)
				}
			}
		})
	}
}

func TestExtractMetadata(t *testing.T) {
	tests := []struct {
		name          string
		topicArn      string
		wantRegion    string
		wantAccountID string
		wantTopicName string
	}{
		{
			name:          "standard ARN",
			topicArn:      "arn:aws:sns:us-east-1:123456789012:prod-alerts",
			wantRegion:    "us-east-1",
			wantAccountID: "123456789012",
			wantTopicName: "prod-alerts",
		},
		{
			name:          "different region",
			topicArn:      "arn:aws:sns:eu-west-1:999888777666:staging-notifications",
			wantRegion:    "eu-west-1",
			wantAccountID: "999888777666",
			wantTopicName: "staging-notifications",
		},
		{
			name:          "topic with dashes",
			topicArn:      "arn:aws:sns:ap-southeast-2:111222333444:my-app-errors",
			wantRegion:    "ap-southeast-2",
			wantAccountID: "111222333444",
			wantTopicName: "my-app-errors",
		},
		{
			name:          "invalid ARN",
			topicArn:      "invalid-arn",
			wantRegion:    "",
			wantAccountID: "",
			wantTopicName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &Message{TopicArn: tt.topicArn}
			msg.extractMetadata()

			if msg.Region != tt.wantRegion {
				t.Errorf("Region = %v, want %v", msg.Region, tt.wantRegion)
			}
			if msg.AccountID != tt.wantAccountID {
				t.Errorf("AccountID = %v, want %v", msg.AccountID, tt.wantAccountID)
			}
			if msg.TopicName != tt.wantTopicName {
				t.Errorf("TopicName = %v, want %v", msg.TopicName, tt.wantTopicName)
			}
		})
	}
}

func TestMessageTypeCheckers(t *testing.T) {
	tests := []struct {
		name                      string
		msgType                   MessageType
		wantIsNotification        bool
		wantIsSubscriptionConfirm bool
		wantIsUnsubscribeConfirm  bool
	}{
		{
			name:                      "notification message",
			msgType:                   MessageTypeNotification,
			wantIsNotification:        true,
			wantIsSubscriptionConfirm: false,
			wantIsUnsubscribeConfirm:  false,
		},
		{
			name:                      "subscription confirmation",
			msgType:                   MessageTypeSubscriptionConfirm,
			wantIsNotification:        false,
			wantIsSubscriptionConfirm: true,
			wantIsUnsubscribeConfirm:  false,
		},
		{
			name:                      "unsubscribe confirmation",
			msgType:                   MessageTypeUnsubscribeConfirm,
			wantIsNotification:        false,
			wantIsSubscriptionConfirm: false,
			wantIsUnsubscribeConfirm:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &Message{Type: tt.msgType}

			if got := msg.IsNotification(); got != tt.wantIsNotification {
				t.Errorf("IsNotification() = %v, want %v", got, tt.wantIsNotification)
			}
			if got := msg.IsSubscriptionConfirmation(); got != tt.wantIsSubscriptionConfirm {
				t.Errorf("IsSubscriptionConfirmation() = %v, want %v", got, tt.wantIsSubscriptionConfirm)
			}
			if got := msg.IsUnsubscribeConfirmation(); got != tt.wantIsUnsubscribeConfirm {
				t.Errorf("IsUnsubscribeConfirmation() = %v, want %v", got, tt.wantIsUnsubscribeConfirm)
			}
		})
	}
}
