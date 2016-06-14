package uploader

import (
	"bytes"
	"errors"
	"fmt"
	"path"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/cloudflare/complainer"
)

func init() {
	var (
		accessKey *string
		secretKey *string
		region    *string
		bucket    *string
		timeout   *time.Duration
	)

	registerMaker("s3aws", Maker{
		RegisterFlags: func() {
			accessKey = s3StringFlag("s3aws.access_key", "S3_ACCESS_KEY", "access key for s3")
			secretKey = s3StringFlag("s3aws.secret_key", "S3_SECRET_KEY", "secret key for s3")
			region = s3StringFlag("s3aws.region", "S3_REGION", "s3 region to use")
			bucket = s3StringFlag("s3aws.bucket", "S3_BUCKET", "s3 bucket to use")
			timeout = s3TimeoutFlag("s3aws.timeout")
		},

		Make: func() (Uploader, error) {
			return newS3AwsUploader(*accessKey, *secretKey, *region, *bucket, *timeout)
		},
	})
}

type s3AwsUploader struct {
	s3      *s3.S3
	bucket  string
	timeout time.Duration
}

func newS3AwsUploader(accessKey, secretKey, region, bucket string, timeout time.Duration) (*s3AwsUploader, error) {
	if accessKey == "" || secretKey == "" || region == "" || bucket == "" {
		return nil, errors.New("s3 configuration is incomplete")
	}

	return &s3AwsUploader{
		s3: s3.New(session.New(&aws.Config{
			Region:      aws.String(region),
			Credentials: credentials.NewStaticCredentials(accessKey, secretKey, ""),
		})),
		bucket:  bucket,
		timeout: timeout,
	}, nil
}

func (u *s3AwsUploader) Upload(failure complainer.Failure, stdoutURL, stderrURL string) (string, string, error) {
	prefix := fmt.Sprintf("complainer/%s/%s-%s", failure.Name, failure.Finished.Format(time.RFC3339), failure.ID)

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
