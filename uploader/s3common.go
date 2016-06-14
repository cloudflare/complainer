package uploader

import (
	"flag"
	"os"
	"time"
)

func s3StringFlag(name, env, help string) *string {
	return flag.String(name, os.Getenv(env), help)
}

func s3TimeoutFlag(name string) *time.Duration {
	defaultTimeout := time.Hour * 24 * 7
	if os.Getenv("S3_TIMEOUT") != "" {
		defaultTimeout, _ = time.ParseDuration(os.Getenv("S3_TIMEOUT"))
	}

	return flag.Duration(name, defaultTimeout, "timeout for signed s3 urls")
}
