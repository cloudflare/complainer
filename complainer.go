package complainer

import (
	"fmt"
	"time"
)

// Failure represents a failed Mesos task
type Failure struct {
	ID        string
	Name      string
	Slave     string
	Framework string
	Image     string
	State     string
	Started   time.Time
	Finished  time.Time
	Labels    map[string]string
}

func (f Failure) String() string {
	return fmt.Sprintf("%s (%s) from %s", f.Name, f.ID, f.Slave)
}
