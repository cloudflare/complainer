package reporter

import (
	"bytes"
	"text/template"

	"fmt"
	"github.com/cloudflare/complainer"
	log "github.com/sirupsen/logrus"
)

func fillTemplate(failure complainer.Failure, config ConfigProvider, stdoutURL, stderrURL, format string) (string, error) {
	logger := log.WithFields(log.Fields{"module": "reporter/template", "func": "fillTemplate"})

	logger.Debugf("Creating template: %s", format)
	tmpl, err := template.New("").Funcs(map[string]interface{}{
		"config": config,
	}).Parse(format)

	if err != nil {
		return "", fmt.Errorf("Failed to create template %s: %s", format, err)
	}

	buf := bytes.NewBuffer([]byte{})

	logger.Debug("Executing template")
	err = tmpl.Execute(buf, map[string]interface{}{
		"nl":        "\n",
		"config":    config,
		"failure":   failure,
		"stdoutURL": stdoutURL,
		"stderrURL": stderrURL,
	})

	if err != nil {
		return "", fmt.Errorf("Error executing template: %s", err)
	}

	return string(buf.Bytes()), nil
}
