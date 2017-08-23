package uploader

import (
	"bytes"
	"errors"
	"path"
	"text/template"
	"time"

	"github.com/cloudflare/complainer"
	"github.com/cloudflare/complainer/flags"
	"github.com/goamz/goamz/aws"
	"github.com/goamz/goamz/s3"
)

func init() {
	var (
		accessKey *string
		secretKey *string
		endpoint  *string
		bucket    *string
		prefix    *string
		timeout   *time.Duration
	)

	registerMaker("s3goamz", Maker{
		RegisterFlags: func() {
			accessKey = flags.String("s3goamz.access_key", "S3_ACCESS_KEY", "", "access key for s3")
			secretKey = flags.String("s3goamz.secret_key", "S3_SECRET_KEY", "", "secret key for s3")
			endpoint = flags.String("s3goamz.endpoint", "S3_ENDPOINT", "", "s3 endpoint (ex: https://complainer.s3.example.com)")
			bucket = flags.String("s3goamz.bucket", "S3_BUCKET", "", "s3 bucket to use")
			prefix = flags.String("s3goamz.prefix", "S3_PREFIX", "complainer/{{ .failure.Finished.UTC.Format \"2006-01-02\" }}/{{ .failure.Name }}/{{ .failure.Finished.UTC.Format \"2006-01-02T15:04:05.000\" }}-{{ .failure.ID }}", "s3 path template to use")
			timeout = flags.Duration("s3goamz.timeout", "S3_TIMEOUT", time.Hour*24*7, "timeout for signed s3 urls")
		},

		Make: func() (Uploader, error) {
			return newS3Uploader(*accessKey, *secretKey, *endpoint, *bucket, *prefix, *timeout)
		},
	})
}

type s3Uploader struct {
	bucket  *s3.Bucket
	timeout time.Duration
	prefix  *template.Template
}

func newS3Uploader(accessKey, secretKey, endpoint, bucket, prefix string, timeout time.Duration) (*s3Uploader, error) {
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

	tmpl, err := template.New("").Parse(prefix)
	if err != nil {
		return nil, err
	}

	return &s3Uploader{
		bucket:  s3.New(auth, region).Bucket(bucket),
		timeout: timeout,
		prefix:  tmpl,
	}, nil
}

func (u *s3Uploader) Upload(failure complainer.Failure, stdoutURL, stderrURL string) (string, string, error) {
	buf := bytes.NewBuffer([]byte{})
	_ = u.prefix.Execute(buf, map[string]interface{}{"failure": failure}) // TODO: absent error handling
	prefix := string(buf.Bytes())

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
