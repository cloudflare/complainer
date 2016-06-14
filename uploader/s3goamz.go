package uploader

import (
	"errors"
	"fmt"
	"path"
	"time"

	"github.com/cloudflare/complainer"
	"github.com/goamz/goamz/aws"
	"github.com/goamz/goamz/s3"
)

func init() {
	var (
		accessKey *string
		secretKey *string
		endpoint  *string
		bucket    *string
		timeout   *time.Duration
	)

	registerMaker("s3", Maker{
		RegisterFlags: func() {
			accessKey = s3StringFlag("s3goamz.access_key", "S3_ACCESS_KEY", "access key for s3")
			secretKey = s3StringFlag("s3goamz.secret_key", "S3_SECRET_KEY", "secret key for s3")
			endpoint = s3StringFlag("s3goamz.endpoint", "S3_ENDPOINT", "s3 endpoint (ex: https://complainer.s3.example.com)")
			bucket = s3StringFlag("s3goamz.bucket", "S3_BUCKET", "s3 bucket to use")
			timeout = s3TimeoutFlag("s3goamz.timeout")
		},

		Make: func() (Uploader, error) {
			return newS3Uploader(*accessKey, *secretKey, *endpoint, *bucket, *timeout)
		},
	})
}

type s3Uploader struct {
	bucket  *s3.Bucket
	timeout time.Duration
}

func newS3Uploader(accessKey, secretKey, endpoint, bucket string, timeout time.Duration) (*s3Uploader, error) {
	if accessKey == "" || secretKey == "" || endpoint == "" || bucket == "" {
		return nil, errors.New("s3 configuration is incomplete")
	}

	auth, err := aws.GetAuth(accessKey, secretKey, "", time.Time{})
	if err != nil {
		return nil, err
	}

	region := aws.Region{
		S3BucketEndpoint: endpoint,
	}

	return &s3Uploader{
		bucket:  s3.New(auth, region).Bucket(bucket),
		timeout: timeout,
	}, nil
}

func (u *s3Uploader) Upload(failure complainer.Failure, stdoutURL, stderrURL string) (string, string, error) {
	prefix := fmt.Sprintf("complainer/%s/%s-%s", failure.Name, failure.Finished.Format(time.RFC3339), failure.ID)

	stdout, err := download(stdoutURL)
	if err != nil {
		return "", "", err
	}

	stdoutPath := path.Join(prefix, "stdout")
	err = u.bucket.Put(stdoutPath, stdout, "text/plain", s3.Private, s3.Options{})
	if err != nil {
		return "", "", err
	}

	stderr, err := download(stderrURL)
	if err != nil {
		return "", "", err
	}

	stderrPath := path.Join(prefix, "stderr")
	err = u.bucket.Put(stderrPath, stderr, "text/plain", s3.Private, s3.Options{})
	if err != nil {
		return "", "", err
	}

	expires := time.Now().Add(u.timeout)

	return u.bucket.SignedURL(stdoutPath, expires), u.bucket.SignedURL(stderrPath, expires), nil
}
