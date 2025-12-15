package app

import (
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/cruxstack/aws-sns-slackbot/internal/config"
	"github.com/cruxstack/aws-sns-slackbot/internal/sns"
)

// fakeSNSEvent returns a sample SNS Lambda event for testing.
func fakeSNSEvent() events.SNSEvent {
	return events.SNSEvent{
		Records: []events.SNSEventRecord{
			{
				EventSource:          "aws:sns",
				EventVersion:         "1.0",
				EventSubscriptionArn: "arn:aws:sns:us-east-1:123456789012:test-topic:12345678-1234-1234-1234-123456789012",
				SNS: events.SNSEntity{
					Type:      "Notification",
					MessageID: "95df01b4-ee98-5cb9-9903-4c221d41eb5e",
					TopicArn:  "arn:aws:sns:us-east-1:123456789012:prod-alerts",
					Subject:   "ERROR: Application Failure",
					Message:   `{"service":"api","error":"database connection timeout","severity":"critical"}`,
					Timestamp: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
				},
			},
		},
	}
}

// fakeSNSEventMultiple returns an SNS event with multiple records for testing.
func fakeSNSEventMultiple() events.SNSEvent {
	return events.SNSEvent{
		Records: []events.SNSEventRecord{
			{
				EventSource: "aws:sns",
				SNS: events.SNSEntity{
					Type:      "Notification",
					MessageID: "msg-1",
					TopicArn:  "arn:aws:sns:us-east-1:123456789012:alerts-critical",
					Subject:   "Critical Alert",
					Message:   "System failure detected",
					Timestamp: time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
				},
			},
			{
				EventSource: "aws:sns",
				SNS: events.SNSEntity{
					Type:      "Notification",
					MessageID: "msg-2",
					TopicArn:  "arn:aws:sns:us-east-1:123456789012:alerts-warning",
					Subject:   "Warning Alert",
					Message:   "High CPU usage",
					Timestamp: time.Date(2025, 1, 15, 10, 5, 0, 0, time.UTC),
				},
			},
		},
	}
}

// fakeSNSEventSubscriptionConfirmation returns a subscription confirmation event.
func fakeSNSEventSubscriptionConfirmation() events.SNSEvent {
	return events.SNSEvent{
		Records: []events.SNSEventRecord{
			{
				EventSource: "aws:sns",
				SNS: events.SNSEntity{
					Type:      "SubscriptionConfirmation",
					MessageID: "sub-conf-123",
					TopicArn:  "arn:aws:sns:us-east-1:123456789012:test-topic",
					Subject:   "",
					Message:   "You have chosen to subscribe to the topic...",
					Timestamp: time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC),
				},
			},
		},
	}
}

// fakeSNSMessage returns a sample parsed SNS message for testing.
func fakeSNSMessage() *sns.Message {
	return &sns.Message{
		Type:      sns.MessageTypeNotification,
		MessageID: "95df01b4-ee98-5cb9-9903-4c221d41eb5e",
		TopicArn:  "arn:aws:sns:us-east-1:123456789012:prod-alerts",
		Subject:   "ERROR: Application Failure",
		Message:   `{"service":"api","error":"database connection timeout","severity":"critical"}`,
		Timestamp: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
		Region:    "us-east-1",
		AccountID: "123456789012",
		TopicName: "prod-alerts",
	}
}

// fakeSNSMessageWithLongContent returns a message with a long content for truncation testing.
func fakeSNSMessageWithLongContent() *sns.Message {
	longMessage := ""
	for i := 0; i < 1000; i++ {
		longMessage += "This is a very long message that will be truncated. "
	}

	return &sns.Message{
		Type:      sns.MessageTypeNotification,
		MessageID: "long-msg-123",
		TopicArn:  "arn:aws:sns:us-east-1:123456789012:verbose-topic",
		Subject:   "Long Message Test",
		Message:   longMessage,
		Timestamp: time.Date(2025, 1, 15, 11, 0, 0, 0, time.UTC),
		Region:    "us-east-1",
		AccountID: "123456789012",
		TopicName: "verbose-topic",
	}
}

// fakeSNSMessageJSON returns a message with JSON content for formatting tests.
func fakeSNSMessageJSON() *sns.Message {
	return &sns.Message{
		Type:      sns.MessageTypeNotification,
		MessageID: "json-msg-123",
		TopicArn:  "arn:aws:sns:us-west-2:987654321098:api-events",
		Subject:   "API Event",
		Message:   `{"requestId":"abc123","method":"POST","path":"/api/users","status":201,"duration":125}`,
		Timestamp: time.Date(2025, 1, 15, 13, 0, 0, 0, time.UTC),
		Region:    "us-west-2",
		AccountID: "987654321098",
		TopicName: "api-events",
	}
}

// fakeRouteConfigs returns sample route configurations for testing.
func fakeRouteConfigs() []config.RouteConfig {
	routes := []config.RouteConfig{
		{
			Name:           "production-alerts",
			SNSTopicArn:    "arn:aws:sns:us-east-1:123456789012:prod-alerts",
			SlackChannelID: "C01234PROD",
			Enabled:        true,
		},
		{
			Name:               "staging-pattern",
			SNSTopicArnPattern: "arn:aws:sns:.*:.*:staging-.*",
			SlackChannelID:     "C01234STAGING",
			Enabled:            true,
		},
		{
			Name:                  "critical-errors",
			SNSTopicArnPattern:    "arn:aws:sns:.*:.*:.*-alerts",
			MessageSubjectPattern: "(?i)(error|critical|fatal)",
			SlackChannelID:        "C01234CRITICAL",
			Enabled:               true,
		},
		{
			Name:           "disabled-route",
			SNSTopicArn:    "arn:aws:sns:us-east-1:123456789012:disabled",
			SlackChannelID: "C01234DISABLED",
			Enabled:        false,
		},
	}

	// Compile patterns
	for i := range routes {
		_ = routes[i].CompilePatterns()
	}

	return routes
}

// fakeRouteConfigsEmpty returns empty route configurations for testing.
func fakeRouteConfigsEmpty() []config.RouteConfig {
	return []config.RouteConfig{}
}

// fakeSNSEventCloudWatchAlarm returns an SNS event with CloudWatch alarm for testing.
func fakeSNSEventCloudWatchAlarm() events.SNSEvent {
	alarmMessage := `{
  "AlarmName": "High-CPU-Utilization",
  "AlarmDescription": "CPU utilization exceeded 80%",
  "AWSAccountId": "123456789012",
  "NewStateValue": "ALARM",
  "NewStateReason": "Threshold Crossed: 1 datapoint [85.0] was greater than the threshold (80.0).",
  "StateChangeTime": "2025-01-15T10:30:45.123Z",
  "Region": "us-east-1",
  "OldStateValue": "OK",
  "Trigger": {
    "MetricName": "CPUUtilization",
    "Namespace": "AWS/EC2",
    "StatisticType": "Statistic",
    "Statistic": "AVERAGE",
    "Unit": "Percent",
    "Dimensions": [
      {
        "name": "InstanceId",
        "value": "i-1234567890abcdef0"
      }
    ],
    "Period": 300,
    "EvaluationPeriods": 1,
    "ComparisonOperator": "GreaterThanThreshold",
    "Threshold": 80.0
  }
}`

	return events.SNSEvent{
		Records: []events.SNSEventRecord{
			{
				EventSource: "aws:sns",
				SNS: events.SNSEntity{
					Type:      "Notification",
					MessageID: "alarm-msg-123",
					TopicArn:  "arn:aws:sns:us-east-1:123456789012:cloudwatch-alarms",
					Subject:   "ALARM: \"High-CPU-Utilization\" in US East (N. Virginia)",
					Message:   alarmMessage,
					Timestamp: time.Date(2025, 1, 15, 10, 30, 45, 0, time.UTC),
				},
			},
		},
	}
}
