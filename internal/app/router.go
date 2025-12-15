package app

import (
	"log/slog"

	"github.com/cruxstack/aws-sns-slackbot/internal/config"
)

// Router handles routing of SNS messages to Slack channels
type Router struct {
	routes          []config.RouteConfig
	defaultChannel  string
	fallbackEnabled bool
	logger          *slog.Logger
}

// NewRouter creates a new message router
func NewRouter(routes []config.RouteConfig, defaultChannel string, fallbackEnabled bool, logger *slog.Logger) *Router {
	return &Router{
		routes:          routes,
		defaultChannel:  defaultChannel,
		fallbackEnabled: fallbackEnabled,
		logger:          logger,
	}
}

// RouteResult contains the routing decision result
type RouteResult struct {
	ChannelID    string
	RouteName    string
	Matched      bool
	UsedFallback bool
}

// FindChannel finds the appropriate Slack channel for a message
// Returns the channel ID and whether a route was matched
// Implements first-match-wins logic: routes are evaluated in order
func (r *Router) FindChannel(topicArn, subject string) RouteResult {
	// Iterate through routes in order (first match wins)
	for _, route := range r.routes {
		if !route.Enabled {
			continue
		}

		// Check topic ARN match (exact or pattern)
		topicMatches := false
		if route.SNSTopicArn != "" {
			// Exact match
			topicMatches = (route.SNSTopicArn == topicArn)
			if topicMatches {
				r.logger.Debug("route matched by exact topic ARN",
					"route", route.Name,
					"topic_arn", topicArn)
			}
		} else if route.SNSTopicArnPattern != "" && route.TopicRegex() != nil {
			// Pattern match
			topicMatches = route.TopicRegex().MatchString(topicArn)
			if topicMatches {
				r.logger.Debug("route matched by topic ARN pattern",
					"route", route.Name,
					"topic_arn", topicArn,
					"pattern", route.SNSTopicArnPattern)
			}
		}

		if !topicMatches {
			continue
		}

		// Check optional subject filter
		if route.MessageSubjectPattern != "" && route.SubjectRegex() != nil {
			if !route.SubjectRegex().MatchString(subject) {
				r.logger.Debug("route topic matched but subject didn't",
					"route", route.Name,
					"subject", subject,
					"pattern", route.MessageSubjectPattern)
				continue // Topic matched but subject didn't
			}
		}

		// FIRST MATCH - return immediately
		r.logger.Info("message routed to channel",
			"route", route.Name,
			"channel", route.SlackChannelID,
			"topic_arn", topicArn)

		return RouteResult{
			ChannelID:    route.SlackChannelID,
			RouteName:    route.Name,
			Matched:      true,
			UsedFallback: false,
		}
	}

	// No routes matched - check fallback
	if r.fallbackEnabled && r.defaultChannel != "" {
		r.logger.Info("no routes matched, using fallback channel",
			"channel", r.defaultChannel,
			"topic_arn", topicArn)

		return RouteResult{
			ChannelID:    r.defaultChannel,
			RouteName:    "fallback",
			Matched:      false,
			UsedFallback: true,
		}
	}

	// No match and no fallback
	r.logger.Warn("no routes matched and fallback is disabled, message will be skipped",
		"topic_arn", topicArn)

	return RouteResult{
		Matched: false,
	}
}
