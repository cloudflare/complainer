package reporter

import (
	"errors"
	"net/url"

	"github.com/cloudflare/complainer"
	"github.com/cloudflare/complainer/flags"
	"github.com/tbruyelle/hipchat-go/hipchat"
)

func init() {
	var (
		baseURL *string
		token   *string
		room    *string
		format  *string
	)

	registerMaker("hipchat", Maker{
		RegisterFlags: func() {
			baseURL = flags.String("hipchat.base_url", "HIPCHAT_BASE_URL", "https://api.hipchat.com/v2/", "default hipchat base url")
			token = flags.String("hipchat.token", "HIPCHAT_TOKEN", "", "default hipchat token")
			room = flags.String("hipchat.room", "HIPCHAT_ROOM", "", "default hipchat room")
			format = flags.String("hipchat.format", "HIPCHAT_FORMAT", "Task {{ .failure.Name }} ({{ .failure.ID }}) died with status {{ .failure.State }} [<a href=\"{{ .stdoutURL }}\">stdout</a>, <a href=\"{{ .stderrURL }}\">stderr</a>]", "log format")
		},

		Make: func() (Reporter, error) {
			return newHipchatReporter(*baseURL, *token, *room, *format), nil
		},
	})
}

type hipchatReporter struct {
	identity hipchatClientIdentity
	room     string
	clients  map[hipchatClientIdentity]*hipchat.Client
	format   string
}

func newHipchatReporter(baseURL, token, room, format string) *hipchatReporter {
	return &hipchatReporter{
		identity: hipchatClientIdentity{
			baseURL: baseURL,
			token:   token,
		},
		room:    room,
		clients: map[hipchatClientIdentity]*hipchat.Client{},
		format:  format,
	}
}

func (h *hipchatReporter) client(baseURL, token string) (*hipchat.Client, error) {
	if token == "" {
		return nil, errors.New("hipchat token is empty")
	}

	identity := hipchatClientIdentity{
		baseURL: baseURL,
		token:   token,
	}

	if client, ok := h.clients[identity]; ok {
		return client, nil
	}

	client := hipchat.NewClient(token)
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	client.BaseURL = parsedURL

	h.clients[identity] = client

	return client, nil
}

func (h *hipchatReporter) Report(failure complainer.Failure, config ConfigProvider, stdoutURL string, stderrURL string) error {
	baseURL := config("base_url")
	if baseURL == "" {
		baseURL = h.identity.baseURL
	}

	token := config("token")
	if token == "" {
		token = h.identity.token
	}

	room := config("room")
	if room == "" {
		room = h.room
	}

	if baseURL == "" || token == "" || room == "" {
		return nil
	}

	client, err := h.client(baseURL, token)
	if err != nil {
		return err
	}

	message, err := fillTemplate(failure, config, stdoutURL, stderrURL, h.format)
	if err != nil {
		return err
	}

	resp, err := client.Room.Notification(room, &hipchat.NotificationRequest{
		MessageFormat: "html",
		Color:         "red",
		Notify:        true,
		Message:       message,
	})

	if err != nil {
		defer func() {
			if resp != nil {
				_ = resp.Body.Close()
			}
		}()
	}

	return err
}

type hipchatClientIdentity struct {
	baseURL string
	token   string
}
