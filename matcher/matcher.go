package matcher

import (
	"regexp"
)

// FailureMatcher is responsible for filtering out undesired Failures for reporting
type FailureMatcher interface {
	Match(string) bool
}

// NoopMatcher ...
type NoopMatcher struct{}

// Match ...
func (c *NoopMatcher) Match(_ string) bool { return true }

// RegexMatcher ...
type RegexMatcher struct {
	Whitelist []*regexp.Regexp
	Blacklist []*regexp.Regexp
}

// Match ...
func (r *RegexMatcher) Match(name string) bool {
	for _, regex := range r.Blacklist {
		if regex.MatchString(name) {
			return false
		}
	}
	for _, regex := range r.Whitelist {
		if regex.MatchString(name) {
			return true
		}
	}
	return len(r.Whitelist) == 0
}
