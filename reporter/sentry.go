package reporter

import (
	"fmt"
	"time"

	"github.com/cloudflare/complainer"
	"github.com/cloudflare/complainer/flags"
	"github.com/getsentry/raven-go"
)

func init() {
	var (
		dsn *string
	)

	registerMaker("sentry", Maker{
		RegisterFlags: func() {
			dsn = flags.String("sentry.dsn", "SENTRY_DSN", "", "sentry dsn")
		},

		Make: func() (Reporter, error) {
			return newSentryReporter(*dsn), nil
		},
	})
}

type sentryReporter struct {
	dsn     string
	clients map[string]*raven.Client
}

func newSentryReporter(dsn string) *sentryReporter {
	return &sentryReporter{
		dsn:     dsn,
		clients: map[string]*raven.Client{},
	}
}

func (s *sentryReporter) client(dsn string) (*raven.Client, error) {
	if client, ok := s.clients[dsn]; ok {
		return client, nil
	}

	client, err := raven.New(dsn)
	if err != nil {
		return client, err
	}

	s.clients[dsn] = client

	return client, nil
}

func (s *sentryReporter) Report(failure complainer.Failure, config ConfigProvider, stdoutURL string, stderrURL string) error {
	dsn := config("dsn")
	if dsn == "" {
		dsn = s.dsn
	}

	if dsn == "" {
		return nil
	}

	client, err := s.client(dsn)
	if err != nil {
		return err
	}

	extra := map[string]interface{}{
		"task.id":          failure.ID,
		"timings.lifetime": failure.Finished.Sub(failure.Started).String(),
		"timings.started":  failure.Started.Format(time.RFC3339),
		"timings.finished": failure.Finished.Format(time.RFC3339),
		"logs.stdout":      stdoutURL,
		"logs.stderr":      stderrURL,
		"framework.name":   failure.Framework,
		"container.image":  failure.Image,
	}

	for k, v := range failure.Labels {
		extra[fmt.Sprintf("labels.%s", k)] = v
	}

	packet := &raven.Packet{
		ServerName: failure.Slave,

		Message: fmt.Sprintf("Task %s died with status %s", failure.Name, failure.State),

		Tags: raven.Tags{
			{
				Key:   "task_state",
				Value: failure.State,
			},
		},

		Extra: extra,
	}

	_, ch := client.Capture(packet, nil)

	return <-ch
}
