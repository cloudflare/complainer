package matcher

import (
	"regexp"
)

// FailureMatcher is responsible for filtering out undesired Failures for reporting
type FailureMatcher interface {
	Match(string) bool
}

type NoopMatcher struct{}

func (c *NoopMatcher) Match(_ string) bool { return true }

type RegexMatcher struct {
	Whitelist []*regexp.Regexp
	Blacklist []*regexp.Regexp
}

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
