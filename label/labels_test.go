package label

import (
	"reflect"
	"testing"
)

func TestTable(t *testing.T) {
	table := []struct {
		complainer string
		labels     map[string]string

		instances map[string][]string
		configs   map[string]map[string]map[string]string
	}{
		{
			complainer: "default",
			labels:     map[string]string{},

			instances: map[string][]string{
				"hipchat": {DefaultInstance},
				"sentry":  {DefaultInstance},
			},
		},
		{
			complainer: "default",
			labels: map[string]string{
				"complainer_sentry_instances": "woo",
			},

			instances: map[string][]string{
				"hipchat": {DefaultInstance},
				"sentry":  {"woo"},
			},
		},
		{
			complainer: "default",
			labels: map[string]string{
				"complainer_sentry_instances": "",
			},

			instances: map[string][]string{
				"hipchat": {DefaultInstance},
				"sentry":  {},
			},
		},
		{
			complainer: "wow",
			labels: map[string]string{
				"complainer_wow_hipchat_instances": "default,sre",

				"complainer_wow_hipchat_room":  "complains",
				"complainer_wow_hipchat_token": "heya",

				"complainer_wow_hipchat_instance_sre_room":  "sre-complains",
				"complainer_wow_hipchat_instance_sre_token": "sosad",

				"complainer_wow_sentry_dsn": "not-dsn",
			},

			instances: map[string][]string{
				"hipchat": {DefaultInstance, "sre"},
				"sentry":  {DefaultInstance},
			},

			configs: map[string]map[string]map[string]string{
				"hipchat": {
					DefaultInstance: {
						"room":  "complains",
						"token": "heya",
					},
					"sre": {
						"room":  "sre-complains",
						"token": "sosad",
					},
				},
				"sentry": {
					DefaultInstance: {
						"dsn": "not-dsn",
					},
				},
			},
		},
	}

	for _, row := range table {
		l := NewLabels(row.complainer, row.labels)

		for r := range row.instances {
			expected := row.instances[r]
			got := l.Instances(r)
			if !reflect.DeepEqual(expected, got) {
				t.Errorf("invalid instances for %s [reporter=%s]; expected: %v, got: %v ", l, r, expected, got)
			}
		}

		if row.configs != nil {
			for r := range row.configs {
				for i := range row.configs[r] {
					for k, expected := range row.configs[r][i] {
						got := l.InstanceLabel(r, i, k)
						if expected != got {
							t.Errorf("invalid config for %s [reporter=%s, instance=%s, key=%s]; expected: %q, got: %q", l, r, i, k, expected, got)
						}
					}
				}
			}
		}
	}
}
