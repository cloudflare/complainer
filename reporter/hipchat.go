package reporter

import (
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"

	"github.com/cloudflare/complainer"
	"github.com/tbruyelle/hipchat-go/hipchat"
)

func init() {
	var (
		baseURL *string
		token   *string
		room    *string
	)

	registerMaker("hipchat", Maker{
		RegisterFlags: func() {
			defaultBaseURL := "https://api.hipchat.com/v2/"
			if os.Getenv("HIPCHAT_BASE_URL") != "" {
				defaultBaseURL = os.Getenv("HIPCHAT_BASE_URL")
			}

			baseURL = flag.String("hipchat.base_url", defaultBaseURL, "default hipchat base url")
			token = flag.String("hipchat.token", os.Getenv("HIPCHAT_TOKEN"), "default hipchat token")
			room = flag.String("hipchat.room", os.Getenv("HIPCHAT_ROOM"), "default hipchat room")
		},

		Make: func() (Reporter, error) {
			return newHipchatReporter(*baseURL, *token, *room), nil
		},
	})
}

type hipchatReporter struct {
	identity hipchatClientIdentity
	room     string
	clients  map[hipchatClientIdentity]*hipchat.Client
}

func newHipchatReporter(baseURL, token, room string) *hipchatReporter {
	return &hipchatReporter{
		identity: hipchatClientIdentity{
			baseURL: baseURL,
			token:   token,
		},
		room:    room,
		clients: map[hipchatClientIdentity]*hipchat.Client{},
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

	resp, err := client.Room.Notification(room, &hipchat.NotificationRequest{
		MessageFormat: "html",
		Color:         "red",
		Notify:        true,
		Message: fmt.Sprintf(
			"Task %s (%s) died with status %s [<a href=%q>stdout</a>, <a href=%q>stderr</a>]",
			failure.Name, failure.Slave, failure.State, stdoutURL, stderrURL,
		),
	})

	if err != nil {
		defer func() {
			_ = resp.Body.Close()
		}()
	}

	return err
}

type hipchatClientIdentity struct {
	baseURL string
	token   string
}
