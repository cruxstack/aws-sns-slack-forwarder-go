# AWS SNS to Slack Forwarder

## What 

A Go-based AWS Lambda function that forwards SNS messages to Slack channels with flexible routing and rich message formatting.

## Why

AWS does not provide a direct integration to Slack. They suggest using a Lambda for forward messages, but the example they provide is extremely limited in functionality. This solutions provides the following robust features:

- **Flexible Routing**: Route messages to different Slack channels based on SNS topic ARN patterns and message subjects
- **Rich Formatting**: Beautiful Slack Block Kit messages with color coding based on severity
- **Smart Configuration**: Support for inline JSON or AWS SSM Parameter Store
- **JSON Pretty-Printing**: Automatically formats JSON messages for readability
- **First-Match Routing**: Routes are evaluated in order, first match wins
- **Fallback Channel**: Optional default channel for messages that don't match any route
- **Message Truncation**: Intelligent truncation for long messages with length indicators

## How It Works

1. Lambda is triggered via SNS notification
2. The Lambda determines the appropriate Slack channel to forward to based on configured routing rules.
3. If a rule is matched, the Lambda forwards the SNS notification to the desired Slack channel.

## Quick Start

### Prerequisites

- Go 1.21 or later
- AWS Account with Lambda permissions
- Slack workspace and bot token with `chat:write` permission
- SNS topics to monitor

### Build

```bash
# Build Lambda function
make build

# Package for deployment
make package-lambda
```

This creates `dist/function.zip` ready for Lambda deployment.

## Deployment

### Lambda Function

1. **Build and package:**
   ```bash
   make package-lambda
   ```

2. **Create Lambda function:**
   ```bash
   aws lambda create-function \
     --function-name sns-slack-forwarder \
     --runtime provided.al2023 \
     --role arn:aws:iam::123456789012:role/lambda-execution-role \
     --handler bootstrap \
     --zip-file fileb://dist/function.zip \
     --timeout 30 \
     --memory-size 256
   ```

3. **Configure environment variables:**
   ```bash
   aws lambda update-function-configuration \
     --function-name sns-slack-forwarder \
     --environment Variables="{
       SLACK_TOKEN=arn:aws:ssm:us-east-1:123456789012:parameter/slack/bot-token,
       SLACK_DEFAULT_CHANNEL=C01234567890,
       SNS_ROUTES_PARAM_ARN=arn:aws:ssm:us-east-1:123456789012:parameter/sns-forwarder/routes,
       FALLBACK_ENABLED=true,
       DEBUG_ENABLED=false
     }"
   ```

4. **Subscribe to SNS topics:**
   ```bash
   aws sns subscribe \
     --topic-arn arn:aws:sns:us-east-1:123456789012:prod-alerts \
     --protocol lambda \
     --notification-endpoint arn:aws:lambda:us-east-1:123456789012:function:sns-slack-forwarder
   ```

5. **Grant SNS permission to invoke Lambda:**
   ```bash
   aws lambda add-permission \
     --function-name sns-slack-forwarder \
     --statement-id sns-invoke \
     --action lambda:InvokeFunction \
     --principal sns.amazonaws.com \
     --source-arn arn:aws:sns:us-east-1:123456789012:prod-alerts
   ```

### IAM Permissions

The Lambda execution role needs:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "logs:CreateLogGroup",
        "logs:CreateLogStream",
        "logs:PutLogEvents"
      ],
      "Resource": "arn:aws:logs:*:*:*"
    },
    {
      "Effect": "Allow",
      "Action": ["ssm:GetParameter"],
      "Resource": [
        "arn:aws:ssm:us-east-1:123456789012:parameter/sns-forwarder/*",
        "arn:aws:ssm:us-east-1:123456789012:parameter/slack/*"
      ]
    }
  ]
}
```

See [examples/iam-policy.json](examples/iam-policy.json) for a complete example.

## Configuration

Configure via environment variables (see [.env.example](.env.example)):

**Required:**
- `SLACK_TOKEN` - Slack bot token (or SSM parameter ARN)
- `SLACK_DEFAULT_CHANNEL` - Default channel ID for fallback

**Routing (choose one):**
- `SNS_ROUTES` - Inline JSON routing configuration
- `SNS_ROUTES_PARAM_ARN` - SSM parameter containing routing config (takes precedence)

**Optional:**
- `FALLBACK_ENABLED` - Send to default channel if no route matches (default: `true`)
- `DEBUG_ENABLED` - Enable debug logging (default: `false`)
- `VALIDATE_SNS_SIGNATURES` - Validate SNS signatures (default: `false`)
- `NOTIFY_ON_SUBSCRIPTION` - Notify when SNS subscription confirmed (default: `false`)

### Routing Configuration

#### Option 1: Inline JSON (Development)

```bash
export SNS_ROUTES='[
  {
    "name": "production-alerts",
    "sns_topic_arn": "arn:aws:sns:us-east-1:123456789012:prod-alerts",
    "slack_channel_id": "C01234567890",
    "enabled": true
  },
  {
    "name": "error-pattern",
    "sns_topic_arn_pattern": ".*-errors$",
    "message_subject_pattern": "ERROR|CRITICAL",
    "slack_channel_id": "C09876543210",
    "enabled": true
  }
]'
```

#### Option 2: SSM Parameter (Production - Recommended)

```bash
# Store routing configuration in SSM
aws ssm put-parameter \
  --name /sns-forwarder/routes \
  --type String \
  --value file://examples/routes.json

# Configure Lambda to use SSM parameter
export SNS_ROUTES_PARAM_ARN=arn:aws:ssm:us-east-1:123456789012:parameter/sns-forwarder/routes
```

**Note:** If `SNS_ROUTES_PARAM_ARN` is set, it takes precedence over `SNS_ROUTES`.

### Route Configuration Fields

Each route can specify:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Unique name for the route |
| `sns_topic_arn` | string | No* | Exact SNS topic ARN to match |
| `sns_topic_arn_pattern` | string | No* | Regex pattern for topic ARN |
| `message_subject_pattern` | string | No | Regex pattern for message subject (optional filter) |
| `slack_channel_id` | string | Yes | Slack channel ID to send to |
| `enabled` | boolean | Yes | Whether this route is active |

\* Must specify either `sns_topic_arn` or `sns_topic_arn_pattern`

### Routing Logic

Routes are evaluated **in order** from top to bottom:

1. Check if route is `enabled`
2. Check if topic ARN matches (exact or pattern)
3. Check if subject matches pattern (if specified)
4. **First matching route wins** - message is sent to that channel
5. If no routes match and `FALLBACK_ENABLED=true`, send to default channel
6. If no routes match and fallback disabled, message is skipped

### Message Formatting

Messages are formatted using Slack Block Kit with:

- **Header**: Topic name with alert emoji
- **Metadata Section**: Subject and timestamp
- **Message Body**: Formatted message content
  - JSON is automatically pretty-printed with 2-space indentation
  - Code blocks for better readability
  - Truncated at 2500 characters with length indicator
- **Footer**: Region, account ID, and message ID
- **Color Sidebar**: Based on subject keywords
  - 🔴 Red: `error`, `critical`, `fatal`, `fail`
  - 🟡 Orange: `warn`, `caution`, `alert`
  - 🟢 Green: `success`, `complete`, `info`
  - 🔵 Blue: Default

## Slack Setup

### Create Slack App

1. Go to [api.slack.com/apps](https://api.slack.com/apps)
2. Click "Create New App" → "From scratch"
3. Name your app and select workspace

### Configure Bot Permissions

Add these OAuth scopes under "OAuth & Permissions":
- `chat:write` - Send messages
- `chat:write.public` - Send to public channels without joining

### Install App

1. Click "Install to Workspace"
2. Copy the "Bot User OAuth Token" (starts with `xoxb-`)
3. Invite bot to channels: `/invite @your-bot-name`

### Get Channel IDs

In Slack, right-click a channel → "View channel details" → Copy the channel ID (starts with `C`)

## Development

### Project Structure

```
aws-sns-slack-forwarder-go/
├── cmd/lambda/          # Lambda entry point
├── internal/
│   ├── app/            # Application logic
│   ├── config/         # Configuration management
│   ├── sns/            # SNS message parsing
│   └── slack/          # Slack client and formatting
├── examples/           # Example configurations
├── Makefile           # Build targets
└── README.md          # This file
```

### Build Commands

```bash
make build    # Build Lambda binary
make package-lambda  # Build and package as zip
make test            # Run tests
make test-verbose    # Run tests with verbose output
make test-coverage   # Run tests with coverage report
make fmt             # Format code
make clean           # Remove build artifacts
make tidy            # Tidy dependencies
```

### Running Tests

The project includes comprehensive unit tests following the patterns from [cruxstack/github-ops-app](https://github.com/cruxstack/github-ops-app).

```bash
# Run all tests
make test

# Run with verbose output
make test-verbose

# Generate coverage report
make test-coverage
# Opens coverage.html in browser
```

**Test Coverage:**
- SNS Parser: 94.7% coverage
- Slack Formatter: 76.0% coverage  
- Router Logic: 31.0% coverage

**Test Structure:**
- `internal/sns/parser_test.go` - SNS message parsing and metadata extraction
- `internal/app/router_test.go` - Routing logic with pattern matching
- `internal/slack/formatter_test.go` - Slack Block Kit formatting and truncation
- `internal/app/testdata.go` - Test data factories (type-safe, no JSON fixtures)

### Local Testing

Create a test event file (`test-event.json`):

```json
{
  "Records": [
    {
      "EventSource": "aws:sns",
      "EventVersion": "1.0",
      "EventSubscriptionArn": "arn:aws:sns:us-east-1:123456789012:test:abc",
      "Sns": {
        "Type": "Notification",
        "MessageId": "abc-123",
        "TopicArn": "arn:aws:sns:us-east-1:123456789012:prod-alerts",
        "Subject": "ERROR: Application failure",
        "Message": "{\"service\": \"api\", \"error\": \"Connection timeout\"}",
        "Timestamp": "2025-01-15T10:30:00.000Z"
      }
    }
  ]
}
```

Test locally:
```bash
export SLACK_TOKEN=xoxb-your-token
export SLACK_DEFAULT_CHANNEL=C01234567890
export SNS_ROUTES='[{"name":"test","sns_topic_arn":"arn:aws:sns:us-east-1:123456789012:prod-alerts","slack_channel_id":"C01234567890","enabled":true}]'

# Build and test
make build
./dist/bootstrap < test-event.json
```

## Troubleshooting

### No messages appearing in Slack

1. **Check Lambda logs:**
   ```bash
   aws logs tail /aws/lambda/sns-slack-forwarder --follow
   ```

2. **Verify bot permissions:**
   - Ensure bot has `chat:write` scope
   - Bot must be invited to channels

3. **Check routing:**
   - Enable `DEBUG_ENABLED=true` to see route matching details
   - Verify topic ARN patterns match your SNS topics

### Configuration errors

1. **Invalid SSM parameter:**
   ```
   Error: failed to fetch routes from SSM parameter
   ```
   - Verify SSM parameter exists
   - Check IAM permissions for `ssm:GetParameter`
   - Ensure parameter ARN is correct format

2. **Route validation failed:**
   ```
   Error: route[0] (test): must specify either sns_topic_arn or sns_topic_arn_pattern
   ```
   - Check JSON syntax in routing configuration
   - Ensure all required fields are present

### Messages truncated

Messages longer than 2500 characters are automatically truncated with a notice. This is a Slack Block Kit limitation. The original message length is shown in the truncation notice.

## Examples

See the [examples/](examples/) directory for:
- [routes.json](examples/routes.json) - Example routing configuration
- [iam-policy.json](examples/iam-policy.json) - Lambda IAM policy

## License

MIT
