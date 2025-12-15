# Lambda Deployment Guide

This guide covers deploying the SNS to Slack forwarder as an AWS Lambda function.

## Prerequisites

- AWS CLI configured with appropriate credentials
- IAM permissions to create Lambda functions and IAM roles
- Slack bot token (see main [README](../../README.md#slack-setup))
- SNS topics to monitor

## Quick Deploy

### 1. Build the Function

```bash
# From project root
make package-lambda
```

This creates `dist/function.zip` containing the Lambda binary.

### 2. Create Execution Role

Create an IAM role for Lambda execution:

```bash
aws iam create-role \
  --role-name sns-slack-forwarder-role \
  --assume-role-policy-document '{
    "Version": "2012-10-17",
    "Statement": [{
      "Effect": "Allow",
      "Principal": {"Service": "lambda.amazonaws.com"},
      "Action": "sts:AssumeRole"
    }]
  }'
```

Attach policies:

```bash
# Basic Lambda execution (CloudWatch Logs)
aws iam attach-role-policy \
  --role-name sns-slack-forwarder-role \
  --policy-arn arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole

# SSM Parameter access (if using SSM for config)
aws iam put-role-policy \
  --role-name sns-slack-forwarder-role \
  --policy-name ssm-parameter-access \
  --policy-document file://../../examples/iam-policy.json
```

### 3. Store Secrets in SSM (Recommended)

```bash
# Store Slack token
aws ssm put-parameter \
  --name /slack/bot-token \
  --type SecureString \
  --value xoxb-your-slack-bot-token \
  --description "Slack bot token for SNS forwarder"

# Store routing configuration
aws ssm put-parameter \
  --name /sns-forwarder/routes \
  --type String \
  --value file://../../examples/routes.json \
  --description "SNS to Slack routing rules"
```

### 4. Create Lambda Function

```bash
aws lambda create-function \
  --function-name sns-slack-forwarder \
  --runtime provided.al2023 \
  --role arn:aws:iam::YOUR_ACCOUNT_ID:role/sns-slack-forwarder-role \
  --handler bootstrap \
  --zip-file fileb://../../dist/function.zip \
  --timeout 30 \
  --memory-size 256 \
  --description "Forwards SNS messages to Slack channels"
```

### 5. Configure Environment Variables

```bash
aws lambda update-function-configuration \
  --function-name sns-slack-forwarder \
  --environment Variables="{
    SLACK_TOKEN=arn:aws:ssm:us-east-1:YOUR_ACCOUNT_ID:parameter/slack/bot-token,
    SLACK_DEFAULT_CHANNEL=C01234567890,
    SNS_ROUTES_PARAM_ARN=arn:aws:ssm:us-east-1:YOUR_ACCOUNT_ID:parameter/sns-forwarder/routes,
    FALLBACK_ENABLED=true,
    DEBUG_ENABLED=false
  }"
```

**Replace:**
- `YOUR_ACCOUNT_ID` with your AWS account ID
- `us-east-1` with your region
- `C01234567890` with your default Slack channel ID

### 6. Subscribe to SNS Topics

For each SNS topic you want to monitor:

```bash
# Subscribe Lambda to SNS topic
aws sns subscribe \
  --topic-arn arn:aws:sns:us-east-1:YOUR_ACCOUNT_ID:prod-alerts \
  --protocol lambda \
  --notification-endpoint arn:aws:lambda:us-east-1:YOUR_ACCOUNT_ID:function:sns-slack-forwarder

# Grant SNS permission to invoke Lambda
aws lambda add-permission \
  --function-name sns-slack-forwarder \
  --statement-id sns-invoke-prod-alerts \
  --action lambda:InvokeFunction \
  --principal sns.amazonaws.com \
  --source-arn arn:aws:sns:us-east-1:YOUR_ACCOUNT_ID:prod-alerts
```

**Note:** You need to run these commands for each SNS topic. Use a unique `--statement-id` for each topic.

### 7. Test the Function

Send a test message to one of your SNS topics:

```bash
aws sns publish \
  --topic-arn arn:aws:sns:us-east-1:YOUR_ACCOUNT_ID:prod-alerts \
  --subject "Test Alert" \
  --message "This is a test message from SNS"
```

Check Slack for the message and CloudWatch Logs for function output:

```bash
aws logs tail /aws/lambda/sns-slack-forwarder --follow
```

## Configuration Updates

### Update Routing Rules

If using SSM for routing configuration, you can update routes without redeploying:

```bash
# Update SSM parameter
aws ssm put-parameter \
  --name /sns-forwarder/routes \
  --type String \
  --value file://../../examples/routes.json \
  --overwrite

# Force Lambda to reload (optional - will reload on next cold start)
aws lambda update-function-configuration \
  --function-name sns-slack-forwarder \
  --description "Updated routing rules"
```

### Update Lambda Code

After making code changes:

```bash
# Build and package
make package-lambda

# Update function code
aws lambda update-function-code \
  --function-name sns-slack-forwarder \
  --zip-file fileb://../../dist/function.zip
```

### Update Environment Variables

```bash
aws lambda update-function-configuration \
  --function-name sns-slack-forwarder \
  --environment Variables="{...}"
```

## Monitoring

### View Logs

```bash
# Tail logs in real-time
aws logs tail /aws/lambda/sns-slack-forwarder --follow

# View recent logs
aws logs tail /aws/lambda/sns-slack-forwarder --since 1h

# Filter for errors
aws logs tail /aws/lambda/sns-slack-forwarder --filter-pattern "ERROR"
```

### CloudWatch Metrics

Monitor these Lambda metrics:
- **Invocations** - Number of times function is invoked
- **Errors** - Number of invocation errors
- **Duration** - Execution time per invocation
- **Throttles** - Number of throttled invocations

### Enable X-Ray Tracing (Optional)

```bash
aws lambda update-function-configuration \
  --function-name sns-slack-forwarder \
  --tracing-config Mode=Active
```

## Troubleshooting

### Permission Errors

**Error:** `User: arn:aws:sts::123456789012:assumed-role/sns-slack-forwarder-role/sns-slack-forwarder is not authorized to perform: ssm:GetParameter`

**Solution:** Ensure IAM role has SSM permissions:
```bash
aws iam put-role-policy \
  --role-name sns-slack-forwarder-role \
  --policy-name ssm-parameter-access \
  --policy-document file://../../examples/iam-policy.json
```

### Function Timeout

**Error:** Task timed out after 3.00 seconds

**Solution:** Increase timeout:
```bash
aws lambda update-function-configuration \
  --function-name sns-slack-forwarder \
  --timeout 30
```

### Memory Issues

**Error:** Runtime exited with error: signal: killed

**Solution:** Increase memory:
```bash
aws lambda update-function-configuration \
  --function-name sns-slack-forwarder \
  --memory-size 512
```

### Messages Not Appearing in Slack

1. Check CloudWatch Logs for errors
2. Enable debug logging: `DEBUG_ENABLED=true`
3. Verify Slack bot token is correct
4. Ensure bot is invited to channels
5. Check routing configuration matches SNS topic ARNs

## Cost Optimization

### Reduce Memory

If your function uses less memory, you can reduce costs:

```bash
# Monitor memory usage in CloudWatch
aws cloudwatch get-metric-statistics \
  --namespace AWS/Lambda \
  --metric-name MemoryUtilization \
  --dimensions Name=FunctionName,Value=sns-slack-forwarder \
  --start-time 2025-01-01T00:00:00Z \
  --end-time 2025-01-15T00:00:00Z \
  --period 3600 \
  --statistics Average

# Adjust memory if needed
aws lambda update-function-configuration \
  --function-name sns-slack-forwarder \
  --memory-size 128
```

### Estimated Costs

Assuming:
- 1000 SNS messages per day
- 100ms average duration
- 256 MB memory

**Monthly cost:** ~$0.20 USD

Free tier includes:
- 1M requests per month
- 400,000 GB-seconds of compute time per month

## Multi-Region Deployment

To deploy in multiple regions:

```bash
# Deploy to us-west-2
aws lambda create-function \
  --region us-west-2 \
  --function-name sns-slack-forwarder \
  --runtime provided.al2023 \
  --role arn:aws:iam::YOUR_ACCOUNT_ID:role/sns-slack-forwarder-role \
  --handler bootstrap \
  --zip-file fileb://../../dist/function.zip

# Configure for us-west-2
aws lambda update-function-configuration \
  --region us-west-2 \
  --function-name sns-slack-forwarder \
  --environment Variables="{...}"
```

## Advanced Configuration

### Reserved Concurrency

Limit concurrent executions to prevent overwhelming Slack API:

```bash
aws lambda put-function-concurrency \
  --function-name sns-slack-forwarder \
  --reserved-concurrent-executions 10
```

### Dead Letter Queue

Configure DLQ for failed invocations:

```bash
# Create DLQ
aws sqs create-queue --queue-name sns-slack-forwarder-dlq

# Configure Lambda DLQ
aws lambda update-function-configuration \
  --function-name sns-slack-forwarder \
  --dead-letter-config TargetArn=arn:aws:sqs:us-east-1:YOUR_ACCOUNT_ID:sns-slack-forwarder-dlq
```

### VPC Configuration

If Slack needs to be accessed through a VPC:

```bash
aws lambda update-function-configuration \
  --function-name sns-slack-forwarder \
  --vpc-config SubnetIds=subnet-abc123,subnet-def456,SecurityGroupIds=sg-abc123
```

**Note:** Ensure NAT Gateway or VPC endpoints for internet access.

## Cleanup

To remove the function and related resources:

```bash
# Delete Lambda function
aws lambda delete-function --function-name sns-slack-forwarder

# Delete IAM role
aws iam detach-role-policy \
  --role-name sns-slack-forwarder-role \
  --policy-arn arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole

aws iam delete-role-policy \
  --role-name sns-slack-forwarder-role \
  --policy-name ssm-parameter-access

aws iam delete-role --role-name sns-slack-forwarder-role

# Delete SSM parameters
aws ssm delete-parameter --name /slack/bot-token
aws ssm delete-parameter --name /sns-forwarder/routes
```
