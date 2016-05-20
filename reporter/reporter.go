package reporter

import (
	"fmt"

	"github.com/cloudflare/complainer"
)

var makers = map[string]Maker{}

// Maker is responsible for making reporter and registering its flags
type Maker struct {
	RegisterFlags func()
	Make          func() (Reporter, error)
}

// RegisterFlags registers flags for all registered makers
func RegisterFlags() {
	for _, rm := range makers {
		rm.RegisterFlags()
	}
}

func registerMaker(name string, reporterMaker Maker) {
	makers[name] = reporterMaker
}

// MakerByName returns registered maker by name
func MakerByName(name string) (Maker, error) {
	if rm, ok := makers[name]; ok {
		return rm, nil
	}

	return Maker{}, fmt.Errorf("unknown reporter maker: %q", name)
}

// Reporter is responsible for reporting failures to external systems
type Reporter interface {
	Report(failure complainer.Failure, config ConfigProvider, stdoutURL, stderrURL string) error
}
