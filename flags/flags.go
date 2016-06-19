package flags

import (
	"flag"
	"os"
	"time"
)

// String registers a flag and returns pointer to the resulting string.
// The default value is passed as fallback and env sets the env variable
// that can overrider the default.
func String(name, env, fallback, help string) *string {
	value := fallback
	if v := os.Getenv(env); v != "" {
		value = v
	}

	return flag.String(name, value, help)
}

// Duration registers a flag and returns pointer to the resulting duration.
// The default value is passed as fallback and env sets the env variable
// that can overrider the default.
func Duration(name, env string, fallback time.Duration, help string) *time.Duration {
	value := fallback
	if v := os.Getenv(env); v != "" {
		value, _ = time.ParseDuration(v)
	}

	return flag.Duration(name, value, help)
}
