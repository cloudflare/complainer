package label

import (
	"fmt"
	"strings"
)

// DefaultInstance is the name of the default reporter instance
const DefaultInstance = "default"

// Labels represent task labels for the specific complainer instance
type Labels struct {
	complainer string
	labels     map[string]string
}

// NewLabels creates labels for the specific complainer instance
func NewLabels(complainer string, labels map[string]string) Labels {
	return Labels{
		complainer: complainer,
		labels:     labels,
	}
}

// Instances returns configured instances of the specific reporter
func (l Labels) Instances(reporter string) []string {
	keys := []string{fmt.Sprintf("complainer_%s_%s_instances", l.complainer, reporter)}

	if l.complainer == DefaultInstance {
		keys = append(keys, fmt.Sprintf("complainer_%s_instances", reporter))
	}

	for _, k := range keys {
		if instances, ok := l.labels[k]; ok {
			if instances == "" {
				return []string{}
			}
			return strings.Split(instances, ",")
		}
	}

	return []string{DefaultInstance}
}

// InstanceLabel returns label value for the specific reporter instance
func (l Labels) InstanceLabel(reporter, instance, name string) string {
	// complainer_default_sentry_instance_default_dsn
	keys := []string{fmt.Sprintf("complainer_%s_%s_instance_%s_%s", l.complainer, reporter, instance, name)}

	if l.complainer == DefaultInstance {
		// complainer_sentry_instance_default_dsn
		keys = append(keys, fmt.Sprintf("complainer_%s_instance_%s_%s", reporter, instance, name))
	}

	if instance == DefaultInstance {
		// complainer_default_sentry_dsn
		keys = append(keys, fmt.Sprintf("complainer_%s_%s_%s", l.complainer, reporter, name))
	}

	if l.complainer == DefaultInstance && instance == DefaultInstance {
		// complainer_sentry_dsn
		keys = append(keys, fmt.Sprintf("complainer_%s_%s", reporter, name))
	}

	for _, k := range keys {
		if l.labels[k] != "" {
			return l.labels[k]
		}
	}

	return ""
}

func (l Labels) String() string {
	return fmt.Sprintf("%s (%v)", l.complainer, l.labels)
}
