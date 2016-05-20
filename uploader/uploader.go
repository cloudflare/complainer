package uploader

import (
	"fmt"

	"github.com/cloudflare/complainer"
)

var makers = map[string]Maker{}

// Maker is responsible for making uploader and registering its flags
type Maker struct {
	RegisterFlags func()
	Make          func() (Uploader, error)
}

// RegisterFlags registers flags for all registered makers
func RegisterFlags() {
	for _, um := range makers {
		um.RegisterFlags()
	}
}

func registerMaker(name string, uploaderMaker Maker) {
	makers[name] = uploaderMaker
}

// MakerByName returns registered maker by name
func MakerByName(name string) (Maker, error) {
	if um, ok := makers[name]; ok {
		return um, nil
	}

	return Maker{}, fmt.Errorf("unknown uploader maker: %q", name)
}

// Uploader is responsible for uploading logs
type Uploader interface {
	Upload(failure complainer.Failure, stdoutURL, stderrURL string) (string, string, error)
}
