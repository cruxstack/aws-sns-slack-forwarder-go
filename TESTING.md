# Testing Guide

This document describes the testing strategy and patterns used in the SNS to Slack forwarder project, modeled after [cruxstack/github-ops-app](https://github.com/cruxstack/github-ops-app).

## Test Philosophy

The testing approach emphasizes:

1. **Table-driven tests** for comprehensive scenario coverage
2. **Type-safe test data factories** instead of JSON fixtures
3. **Unit tests without heavy mocking** - use struct initialization
4. **High coverage of critical paths** - parsing, routing, formatting

## Test Coverage

Current test coverage by package:

| Package | Coverage | Test File | Focus Area |
|---------|----------|-----------|------------|
| `internal/sns` | 94.7% | `parser_test.go` | SNS event parsing, metadata extraction |
| `internal/slack` | 76.0% | `formatter_test.go` | Block Kit formatting, JSON pretty-printing, truncation |
| `internal/app` | 31.0% | `router_test.go` | First-match routing, pattern matching |

## Running Tests

```bash
# Run all tests
make test

# Run with verbose output (see log messages)
make test-verbose

# Generate HTML coverage report
make test-coverage
```

## Test Structure

### SNS Parser Tests (`internal/sns/parser_test.go`)

Tests SNS message parsing from Lambda events:

```go
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
                Records: []events.SNSEventRecord{...},
            },
            wantCount: 1,
            wantErr:   false,
        },
        // more test cases...
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ParseEvent(tt.event)
            // assertions...
        })
    }
}
```

**Tests cover:**
- ✅ Single and multiple SNS records
- ✅ Notification vs subscription confirmation messages
- ✅ ARN metadata extraction (region, account ID, topic name)
- ✅ Message type checking helpers
- ✅ Invalid ARN handling

### Router Tests (`internal/app/router_test.go`)

Tests message routing logic with first-match-wins behavior:

```go
func TestRouterFindChannel(t *testing.T) {
    tests := []struct {
        name            string
        routes          []config.RouteConfig
        topicArn        string
        subject         string
        wantChannelID   string
        wantMatched     bool
    }{
        {
            name: "exact topic ARN match",
            routes: []config.RouteConfig{...},
            topicArn: "arn:aws:sns:us-east-1:123:prod-alerts",
            wantChannelID: "C01234567890",
            wantMatched: true,
        },
        // more test cases...
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            router := NewRouter(...)
            result := router.FindChannel(tt.topicArn, tt.subject)
            // assertions...
        })
    }
}
```

**Tests cover:**
- ✅ Exact topic ARN matching
- ✅ Regex pattern matching for topics
- ✅ Subject pattern filtering
- ✅ First-match-wins behavior (route order matters)
- ✅ Disabled route skipping
- ✅ Fallback channel logic
- ✅ Complex regex patterns (ends-with, starts-with, region-specific)

### Slack Formatter Tests (`internal/slack/formatter_test.go`)

Tests Slack Block Kit message formatting:

```go
func TestDetectSeverityColor(t *testing.T) {
    tests := []struct {
        name    string
        subject string
        want    string  // Hex color code
    }{
        {
            name:    "error keyword",
            subject: "ERROR: Database connection failed",
            want:    "#d32f2f",  // Red
        },
        // more test cases...
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            f := NewFormatter()
            got := f.detectSeverityColor(tt.subject)
            // assertions...
        })
    }
}
```

**Tests cover:**
- ✅ Severity color detection (red/orange/green/blue)
- ✅ JSON detection and pretty-printing with 2-space indent
- ✅ Message truncation at 2500 chars with length indicator
- ✅ Block Kit message structure creation
- ✅ Long message handling

## Test Data Factories

Located in `internal/app/testdata.go`, these functions provide type-safe test data:

```go
// fakeSNSEvent returns a sample SNS Lambda event for testing.
func fakeSNSEvent() events.SNSEvent {
    return events.SNSEvent{
        Records: []events.SNSEventRecord{
            {
                EventSource: "aws:sns",
                SNS: events.SNSEntity{
                    Type:      "Notification",
                    MessageID: "95df01b4-ee98-5cb9-9903-4c221d41eb5e",
                    TopicArn:  "arn:aws:sns:us-east-1:123456789012:prod-alerts",
                    Subject:   "ERROR: Application Failure",
                    Message:   `{"service":"api","error":"database connection timeout"}`,
                    Timestamp: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
                },
            },
        },
    }
}
```

**Available factories:**
- `fakeSNSEvent()` - Standard notification message
- `fakeSNSEventMultiple()` - Multiple records in one event
- `fakeSNSEventSubscriptionConfirmation()` - Subscription confirmation
- `fakeSNSEventCloudWatchAlarm()` - CloudWatch alarm with JSON payload
- `fakeSNSMessage()` - Parsed SNS message
- `fakeSNSMessageWithLongContent()` - For truncation testing
- `fakeSNSMessageJSON()` - JSON content for formatting tests
- `fakeRouteConfigs()` - Sample routing configuration with compiled patterns

**Benefits:**
- Type-safe: Compiler catches errors
- Realistic: Represents actual AWS SDK types
- Maintainable: Single source of truth
- Flexible: Easy to customize per test

## Test Patterns

### Table-Driven Tests

All tests use table-driven patterns for comprehensive coverage:

```go
tests := []struct {
    name     string  // Test case name
    input    Type    // Input parameters
    want     Type    // Expected result
    wantErr  bool    // Expect error?
}{
    {
        name:  "success case",
        input: validInput,
        want:  expectedOutput,
    },
    {
        name:    "error case",
        input:   invalidInput,
        wantErr: true,
    },
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        // test logic
    })
}
```

### No Mocking Framework

Following github-ops-app patterns, tests use **struct initialization** instead of mocking frameworks:

```go
// ✅ DO: Direct struct initialization
router := NewRouter(routes, "default-channel", true, logger)
result := router.FindChannel(topicArn, subject)

// ❌ DON'T: Heavy mocking frameworks
// mock := new(MockRouter)
// mock.On("FindChannel", ...).Return(...)
```

**Why?**
- Simpler code
- Faster tests
- Tests real logic paths
- No mock setup/teardown complexity

## Adding New Tests

### 1. Create Test File

Place test files alongside source files:

```
internal/mypackage/
├── mycode.go
└── mycode_test.go
```

### 2. Use Table-Driven Pattern

```go
func TestMyFunction(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {name: "valid input", input: "test", want: "TEST"},
        {name: "empty input", input: "", wantErr: true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := MyFunction(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("MyFunction() error = %v, wantErr %v", err, tt.wantErr)
            }
            if got != tt.want {
                t.Errorf("MyFunction() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### 3. Add Test Data Factory

If you need complex test data, add a factory to `testdata.go`:

```go
func fakeMyData() *MyType {
    return &MyType{
        Field1: "value1",
        Field2: 123,
        // ... realistic test data
    }
}
```

### 4. Run and Verify

```bash
# Run new tests
go test -v ./internal/mypackage/...

# Check coverage
go test -cover ./internal/mypackage/...

# Update coverage report
make test-coverage
```

## Test Examples

### Testing SNS Message Processing

```go
func TestProcessMessage(t *testing.T) {
    // Use test factory
    msg := fakeSNSMessage()
    
    // Setup router
    routes := fakeRouteConfigs()
    router := NewRouter(routes, "C_DEFAULT", true, logger)
    
    // Test routing
    result := router.FindChannel(msg.TopicArn, msg.Subject)
    
    // Assertions
    if !result.Matched {
        t.Error("expected message to match route")
    }
    if result.ChannelID == "" {
        t.Error("expected non-empty channel ID")
    }
}
```

### Testing JSON Pretty-Printing

```go
func TestJSONFormatting(t *testing.T) {
    formatter := NewFormatter()
    
    // Test with JSON message
    msg := fakeSNSMessageJSON()
    formatted := formatter.formatMessageBody(msg.Message)
    
    // Verify pretty-printing
    if !strings.Contains(formatted, "\n") {
        t.Error("expected newlines in formatted JSON")
    }
    if !strings.Contains(formatted, "  ") {
        t.Error("expected 2-space indentation")
    }
}
```

### Testing Route Precedence

```go
func TestFirstMatchWins(t *testing.T) {
    routes := []config.RouteConfig{
        {
            Name:               "first-route",
            SNSTopicArnPattern: ".*:prod-.*",
            SlackChannelID:     "C_FIRST",
            Enabled:            true,
        },
        {
            Name:               "second-route",
            SNSTopicArnPattern: ".*:prod-.*",  // Same pattern
            SlackChannelID:     "C_SECOND",
            Enabled:            true,
        },
    }
    
    // Compile patterns
    for i := range routes {
        routes[i].CompilePatterns()
    }
    
    router := NewRouter(routes, "C_DEFAULT", true, logger)
    result := router.FindChannel("arn:aws:sns:us-east-1:123:prod-alerts", "")
    
    // First route should win
    if result.ChannelID != "C_FIRST" {
        t.Errorf("expected first route to win, got channel %s", result.ChannelID)
    }
}
```

## Future Testing Enhancements

Potential areas for expansion (following github-ops-app patterns):

1. **Integration Tests with Mock Servers**
   - HTTP mock servers for Slack API
   - Test real Slack SDK against mocks
   - Capture and validate API calls

2. **Scenario-Based Testing**
   - JSON scenario definitions
   - End-to-end workflow tests
   - Multiple message sequences

3. **Config Validation Tests**
   - Test SSM parameter loading
   - Route compilation errors
   - Configuration edge cases

4. **Performance Tests**
   - Large message handling
   - Multiple concurrent routes
   - Memory usage validation

## Continuous Integration

The tests are designed to run in CI/CD pipelines:

```yaml
# Example GitHub Actions workflow
- name: Run Tests
  run: |
    go test -v -race -count=1 ./...
    
- name: Check Coverage
  run: |
    go test -coverprofile=coverage.out ./...
    go tool cover -func=coverage.out
```

## References

- [github-ops-app Testing Patterns](https://github.com/cruxstack/github-ops-app)
- [Go Testing Package](https://pkg.go.dev/testing)
- [Table-Driven Tests in Go](https://go.dev/wiki/TableDrivenTests)
