package uploader

import "github.com/cloudflare/complainer"

func init() {
	registerMaker("noop", Maker{
		RegisterFlags: func() {},

		Make: func() (Uploader, error) {
			return noopUploader{}, nil
		},
	})
}

type noopUploader struct{}

func (n noopUploader) Upload(failure complainer.Failure, stdoutURL, stderrURL string) (string, string, error) {
	return stdoutURL, stderrURL, nil
}
