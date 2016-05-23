package uploader

import (
	"flag"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/cloudflare/complainer"
	"github.com/goamz/goamz/aws"
	"github.com/goamz/goamz/s3"
)

func init() {
	var (
		accessKey      *string
		secretKey      *string
		endpoint       *string
		bucketEndpoint *string
		bucket         *string
		timeout        *time.Duration
	)

	registerMaker("s3", Maker{
		RegisterFlags: func() {
			defaultTimeout := time.Hour * 24 * 30
			if os.Getenv("S3_TIMEOUT") != "" {
				defaultTimeout, _ = time.ParseDuration(os.Getenv("S3_TIMEOUT"))
			}

			accessKey = flag.String("s3.access_key", os.Getenv("S3_ACCESS_KEY"), "access key for s3")
			secretKey = flag.String("s3.secret_key", os.Getenv("S3_SECRET_KEY"), "secret key for s3")
			endpoint = flag.String("s3.endpoint", os.Getenv("S3_ENDPOINT"), "s3 endpoint (ex: https://s3-eu-central-1.amazonaws.com)")
			bucketEndpoint = flag.String("s3.bucket_endpoint", os.Getenv("S3_BUCKET_ENDPOINT"), "s3 bucket endpoint (ex: https://${bucket}.my.cusom.domain)")
			bucket = flag.String("s3.bucket", os.Getenv("S3_BUCKET"), "s3 bucket to use")
			timeout = flag.Duration("s3.timeout", defaultTimeout, "timeout for signed s3 urls") // TODO: infinite?
		},

		Make: func() (Uploader, error) {
			return newS3Uploader(*accessKey, *secretKey, *endpoint, *bucketEndpoint, *bucket, *timeout)
		},
	})
}

type s3Uploader struct {
	bucket  *s3.Bucket
	timeout time.Duration
}

func newS3Uploader(accessKey, secretKey, endpoint, bucketEndpoint, bucket string, timeout time.Duration) (*s3Uploader, error) {
	auth, err := aws.GetAuth(accessKey, secretKey, "", time.Time{})
	if err != nil {
		return nil, err
	}

	region := aws.Region{
		S3Endpoint:       endpoint,
		S3BucketEndpoint: bucketEndpoint, // TODO: simplify/remove?
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
