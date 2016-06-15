package reporter

import (
	"bytes"
	"text/template"

	"github.com/cloudflare/complainer"
)

func fillTemplate(failure complainer.Failure, config ConfigProvider, stdoutURL, stderrURL, format string) (string, error) {
	tmpl, err := template.New("").Funcs(map[string]interface{}{
		"config": config,
	}).Parse(format)
	if err != nil {
		return "", err
	}

	buf := bytes.NewBuffer([]byte{})

	err = tmpl.Execute(buf, map[string]interface{}{
		"nl":        "\n",
		"config":    config,
		"failure":   failure,
		"stdoutURL": stdoutURL,
		"stderrURL": stderrURL,
	})

	return string(buf.Bytes()), err
}
