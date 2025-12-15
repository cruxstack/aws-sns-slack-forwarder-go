package config

import "regexp"

// RouteConfig defines a routing rule for SNS messages to Slack channels
type RouteConfig struct {
	Name                  string `json:"name"`
	SNSTopicArn           string `json:"sns_topic_arn,omitempty"`
	SNSTopicArnPattern    string `json:"sns_topic_arn_pattern,omitempty"`
	MessageSubjectPattern string `json:"message_subject_pattern,omitempty"`
	SlackChannelID        string `json:"slack_channel_id"`
	Enabled               bool   `json:"enabled"`

	// Compiled regex patterns (not exported in JSON)
	topicRegex   *regexp.Regexp
	subjectRegex *regexp.Regexp
}

// TopicRegex returns the compiled topic ARN pattern regex
func (r *RouteConfig) TopicRegex() *regexp.Regexp {
	return r.topicRegex
}

// SubjectRegex returns the compiled subject pattern regex
func (r *RouteConfig) SubjectRegex() *regexp.Regexp {
	return r.subjectRegex
}

// CompilePatterns compiles the regex patterns in the route
func (r *RouteConfig) CompilePatterns() error {
	if r.SNSTopicArnPattern != "" {
		regex, err := regexp.Compile(r.SNSTopicArnPattern)
		if err != nil {
			return err
		}
		r.topicRegex = regex
	}

	if r.MessageSubjectPattern != "" {
		regex, err := regexp.Compile(r.MessageSubjectPattern)
		if err != nil {
			return err
		}
		r.subjectRegex = regex
	}

	return nil
}
