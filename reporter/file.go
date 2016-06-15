package reporter

import (
	"flag"
	"os"

	"github.com/cloudflare/complainer"
)

func init() {
	var (
		file   *string
		format *string
	)

	registerMaker("file", Maker{
		RegisterFlags: func() {
			file = flag.String("file.name", "/dev/stderr", "file to log failures")
			format = flag.String("file.format", "Task {{ .failure.Name }} ({{ .failure.ID }}) died with status {{ .failure.State }}:{{ .nl }}  * {{ .stdoutURL }}{{ .nl }}  * {{ .stderrURL }} ]{{ .nl }}", "log format")
		},

		Make: func() (Reporter, error) {
			return newFileReporter(*file, *format)
		},
	})
}

type fileReporter struct {
	file   *os.File
	format string
}

func newFileReporter(file, format string) (*fileReporter, error) {
	f, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}

	return &fileReporter{
		file:   f,
		format: format,
	}, nil
}

func (f *fileReporter) Report(failure complainer.Failure, config ConfigProvider, stdoutURL string, stderrURL string) error {
	s, err := fillTemplate(failure, config, stdoutURL, stderrURL, f.format)
	if err != nil {
		return err
	}

	_, err = f.file.WriteString(s)
	return err
}
