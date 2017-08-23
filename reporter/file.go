package reporter

import (
	"os"

	log "github.com/sirupsen/logrus"

	"fmt"
	"github.com/cloudflare/complainer"
	"github.com/cloudflare/complainer/flags"
)

func init() {
	var (
		file   *string
		format *string
	)

	registerMaker("file", Maker{
		RegisterFlags: func() {
			file = flags.String("file.name", "FILE_NAME", "/dev/stderr", "file to log failures")
			format = flags.String("file.format", "FILE_FORMAT", "Task {{ .failure.Name }} ({{ .failure.ID }}) died with status {{ .failure.State }}:{{ .nl }}  * {{ .stdoutURL }}{{ .nl }}  * {{ .stderrURL }}{{ .nl }}", "log format")
		},

		Make: func() (Reporter, error) {
			return newFileReporter(*file, *format)
		},
	})
}

type fileReporter struct {
	file   *os.File
	format string
	log    *log.Entry
}

func newFileReporter(file, format string) (*fileReporter, error) {
	f, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return nil, err
	}

	return &fileReporter{
		file:   f,
		format: format,
		log:    log.WithField("module", "reporter/file"),
	}, nil
}

func (f *fileReporter) Report(failure complainer.Failure, config ConfigProvider, stdoutURL string, stderrURL string) error {
	logger := f.log.WithField("func", "Report")

	s, err := fillTemplate(failure, config, stdoutURL, stderrURL, f.format)
	if err != nil {
		return fmt.Errorf("fillTemplate(): %s", err)
	}

	logger.Infof("File reporter: reporting failure of %s", failure.ID)

	_, err = f.file.WriteString(s)
	if err != nil {
		return fmt.Errorf("WriteString(): %s", err)
	}
	return nil
}
