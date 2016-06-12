package reporter

import (
	"flag"
	"os"
	"text/template"

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
			format = flag.String("file.format", "Task {{ .failure.Name }} ({{ .failure.ID }}) died with status {{ .failure.State }}:\n  * {{ .stdoutURL }}\n  * {{ .stderrURL }} ]\n", "log format")
		},

		Make: func() (Reporter, error) {
			return newFileReporter(*file, *format)
		},
	})
}

type fileReporter struct {
	file     *os.File
	template *template.Template
}

func newFileReporter(file, format string) (*fileReporter, error) {
	f, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}

	tmpl, err := template.New("").Parse(format)
	if err != nil {
		return nil, err
	}

	return &fileReporter{
		file:     f,
		template: tmpl,
	}, nil
}

func (f *fileReporter) Report(failure complainer.Failure, config ConfigProvider, stdoutURL string, stderrURL string) error {
	return f.template.Execute(f.file, map[string]interface{}{
		"failure":   failure,
		"stdoutURL": stdoutURL,
		"stderrURL": stderrURL,
	})
}
