package reporter

import "github.com/cloudflare/complainer/label"

// ConfigProvider is a function that returns the value of the config key
type ConfigProvider func(key string) string

// NewConfigProvider returns ConfigProvider implementation based labels
func NewConfigProvider(labels label.Labels, reporter, instance string) ConfigProvider {
	return func(key string) string {
		return labels.InstanceLabel(reporter, instance, key)
	}
}
