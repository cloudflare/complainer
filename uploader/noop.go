package uploader

import (
	"github.com/cloudflare/complainer"
	log "github.com/sirupsen/logrus"
)

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
	log.WithFields(log.Fields{"module": "uploader/noop", "func": "Upload"}).
		Infof("No-op upload: failure ID: %s, stdout: %s, stderr: %s", failure.ID, stdoutURL, stderrURL)

	return stdoutURL, stderrURL, nil
}
