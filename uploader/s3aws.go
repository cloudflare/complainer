package uploader

import (
	"bytes"
	"errors"
	"path"
	"text/template"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/cloudflare/complainer"
	"github.com/cloudflare/complainer/flags"
)

func init() {
	var (
		accessKey *string
		secretKey *string
		region    *string
		bucket    *string
		prefix    *string
		timeout   *time.Duration
	)

	registerMaker("s3aws", Maker{
		RegisterFlags: func() {
			accessKey = flags.String("s3aws.access_key", "S3_ACCESS_KEY", "", "access key for s3")
			secretKey = flags.String("s3aws.secret_key", "S3_SECRET_KEY", "", "secret key for s3")
			region = flags.String("s3aws.region", "S3_REGION", "", "s3 region to use")
			bucket = flags.String("s3aws.bucket", "S3_BUCKET", "", "s3 bucket to use")
			prefix = flags.String("s3aws.prefix", "S3_PREFIX", "complainer/{{ .failure.Finished.UTC.Format \"2006-01-02\" }}/{{ .failure.Name }}/{{ .failure.Finished.UTC.Format \"2006-01-02T15:04:05.000\" }}-{{ .failure.ID }}", "s3 path template to use")
			timeout = flags.Duration("s3aws.timeout", "S3_TIMEOUT", time.Hour*24*7, "timeout for signed s3 urls")
		},

		Make: func() (Uploader, error) {
			return newS3AwsUploader(*accessKey, *secretKey, *region, *bucket, *prefix, *timeout)
		},
	})
}

type s3AwsUploader struct {
	s3      *s3.S3
	bucket  string
	prefix  *template.Template
	timeout time.Duration
}

func newS3AwsUploader(accessKey, secretKey, region, bucket, prefix string, timeout time.Duration) (*s3AwsUploader, error) {
	if accessKey == "" || secretKey == "" || region == "" || bucket == "" {
		return nil, errors.New("s3 configuration is incomplete")
	}

	tmpl, err := template.New("").Parse(prefix)
	if err != nil {
		return nil, err
	}

	return &s3AwsUploader{
		s3: s3.New(session.New(&aws.Config{
			Region:      aws.String(region),
			Credentials: credentials.NewStaticCredentials(accessKey, secretKey, ""),
		})),
		bucket:  bucket,
		timeout: timeout,
		prefix:  tmpl,
	}, nil
}

func (u *s3AwsUploader) Upload(failure complainer.Failure, stdoutURL, stderrURL string) (string, string, error) {
	buf := bytes.NewBuffer([]byte{})
	err := u.prefix.Execute(buf, map[string]interface{}{"failure": failure})
	prefix := string(buf.Bytes())

	stdout, err := download(stdoutURL)
	if err != nil {
		return "", "", err
	}

	signedStdoutURL, err := u.upload(path.Join(prefix, "stdout"), stdout)
	if err != nil {
		return "", "", err
	}

	stderr, err := download(stderrURL)
	if err != nil {
		return "", "", err
	}

	signedStderrURL, err := u.upload(path.Join(prefix, "stderr"), stderr)
	if err != nil {
		return "", "", err
	}

	return signedStdoutURL, signedStderrURL, nil
}

func (u *s3AwsUploader) upload(key string, data []byte) (string, error) {
	_, err := u.s3.PutObject(&s3.PutObjectInput{
		ACL:           aws.String(s3.ObjectCannedACLPrivate),
		Body:          bytes.NewReader(data),
		Bucket:        aws.String(u.bucket),
		ContentType:   aws.String("text/plain"),
		ContentLength: aws.Int64(int64(len(data))),
		Key:           aws.String(key),
	})
	if err != nil {
		return "", err
	}

	r, _ := u.s3.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(u.bucket),
		Key:    aws.String(key),
	})

	return r.Presign(u.timeout)
}
