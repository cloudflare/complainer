package reporter

import (
	"bytes"
	"encoding/json"
	"flag"
	"net/http"
	"net/url"
	"os"

	"github.com/cloudflare/complainer"
)

func init() {
	var (
		hookURL   *string
		username  *string
		channel   *string
		iconEmoji *string
		iconURL   *string
		format    *string
	)

	registerMaker("slack", Maker{
		RegisterFlags: func() {
			hookURL = flag.String("slack.hook_url", os.Getenv("SLACK_HOOK_URL"), "default slack webhook url")
			username = flag.String("slack.username", os.Getenv("SLACK_USERNAME"), "default slack username")
			channel = flag.String("slack.channel", os.Getenv("SLACK_CHANNEL"), "default slack channel")
			iconEmoji = flag.String("slack.icon_emoji", os.Getenv("SLACK_ICON_EMOJI"), "default slack user icon emoji")
			iconURL = flag.String("slack.icon_url", os.Getenv("SLACK_ICON_URL"), "default slack user icon url")
			format = flag.String("slack.format", "Task {{ .failure.Name }} ({{ .failure.ID }}) died with status {{ .failure.State }} [<{{ .stdoutURL }}|stdout>, <{{ .stderrURL }}|stderr>]", "log format")
		},

		Make: func() (Reporter, error) {
			return newSlackReporter(*hookURL, *username, *channel, *iconEmoji, *iconURL, *format)
		},
	})
}

type slackReporter struct {
	hookURL   *url.URL
	channel   string
	username  string
	iconEmoji string
	iconURL   string
	format    string
}

type slackMessage struct {
	Channel   string `json:"channel"`
	Username  string `json:"username"`
	Text      string `json:"text"`
	IconEmoji string `json:"icon_emoji"`
	IconURL   string `json:"icon_url"`
}

func newSlackReporter(hookURL, username, channel, iconEmoji, iconURL, format string) (*slackReporter, error) {
	u, err := url.Parse(hookURL)
	if err != nil {
		return nil, err
	}

	return &slackReporter{
		hookURL:   u,
		username:  username,
		channel:   channel,
		iconEmoji: iconEmoji,
		iconURL:   iconURL,
		format:    format,
	}, nil
}

func (s *slackReporter) Report(failure complainer.Failure, config ConfigProvider, stdoutURL string, stderrURL string) error {
	text, err := fillTemplate(failure, config, stdoutURL, stderrURL, s.format)
	if err != nil {
		return err
	}

	m := &slackMessage{
		Text: text,
	}

	var hookURL *url.URL
	if u := config("hook_url"); len(u) > 0 {
		hookURL, err = url.Parse(u)
		if err != nil {
			return err
		}

	} else {
		hookURL = s.hookURL
	}

	// A hook url is the only required property here.
	// All other properties are optional.
	// But it's a legitimate scenario when some reporters are not fully configured.
	// You don't always want to report all failures to all reporters.
	// If some required parameter is missing, just silently return from Report().
	if hookURL == nil {
		return nil
	}

	// Fill and overwrite configuration values
	s.fillConfigValues(m, config)

	jsonMessage, err := json.Marshal(m)
	if err != nil {
		return err
	}

	body := bytes.NewReader(jsonMessage)
	resp, err := http.Post(hookURL.String(), "application/json", body)
	if err != nil {
		defer func() {
			if resp != nil {
				_ = resp.Body.Close()
			}
		}()
	}

	return err
}

func (s *slackReporter) fillConfigValues(m *slackMessage, config ConfigProvider) {
	// Check the user name overwrite
	if username := config("username"); len(username) > 0 {
		m.Username = username
	} else if len(s.username) > 0 {
		m.Username = s.username
	}

	// Check the channel overwrite
	if channel := config("channel"); len(channel) > 0 {
		m.Channel = channel
	} else if len(s.channel) > 0 {
		m.Channel = s.channel
	}

	// Check the icon emoji overwrite
	if emoji := config("icon_emoji"); len(emoji) > 0 {
		m.IconEmoji = emoji
	} else if len(s.iconEmoji) > 0 {
		m.IconEmoji = s.iconEmoji
	}

	// Check the icon url overwrite
	if iconURL := config("icon_url"); len(iconURL) > 0 {
		m.IconURL = iconURL
	} else if len(s.iconURL) > 0 {
		m.IconURL = s.iconURL
	}
}
