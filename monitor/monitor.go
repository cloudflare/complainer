package monitor

import (
	"fmt"
	"log"
	"net/http"
	"net/http/pprof"
	"sync"
	"time"

	"github.com/cloudflare/complainer"
	"github.com/cloudflare/complainer/label"
	"github.com/cloudflare/complainer/mesos"
	"github.com/cloudflare/complainer/reporter"
	"github.com/cloudflare/complainer/uploader"
)

const (
	// DefaultName is the default name of the complainer instance
	DefaultName = "default"
	// timeout before purging old seen tasks
	timeout = time.Minute
)

// Monitor is responsible for routing failed tasks to the configured reporters
type Monitor struct {
	name      string
	mesos     *mesos.Cluster
	uploader  uploader.Uploader
	reporters map[string]reporter.Reporter
	defaults  bool
	recent    map[string]time.Time
	mu        sync.Mutex
	err       error
}

// NewMonitor creates the new monitor with a name, uploader and reporters
func NewMonitor(name string, cluster *mesos.Cluster, up uploader.Uploader, reporters map[string]reporter.Reporter, defaults bool) *Monitor {
	return &Monitor{
		name:      name,
		mesos:     cluster,
		uploader:  up,
		reporters: reporters,
		defaults:  defaults,
	}
}

// ListenAndServe launches an http server on the requested address.
// The server is responsible for health checks
func (m *Monitor) ListenAndServe(addr string) error {
	mux := http.NewServeMux()

	// health check
	mux.HandleFunc("/health", m.handleHealthCheck)

	// pprof
	mux.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
	mux.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	mux.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	mux.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	mux.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))

	return http.ListenAndServe(addr, mux)
}

func (m *Monitor) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	err := m.err
	m.mu.Unlock()

	if err == nil {
		_, _ = w.Write([]byte("I am mostly okay, thanks.\n"))
		return
	}

	w.WriteHeader(http.StatusInternalServerError)
	_, _ = w.Write([]byte(fmt.Sprintf("Something is fishy: %s\n", err)))
}

// Run does one run across failed tasks and reports any new failures
func (m *Monitor) Run() error {
	failures, err := m.mesos.Failures()
	defer func() {
		m.mu.Lock()
		m.err = err
		m.mu.Unlock()
	}()

	if err != nil {
		return err
	}

	first := false
	if m.recent == nil {
		m.recent = map[string]time.Time{}
		first = true
	}

	for _, failure := range failures {
		if m.checkFailure(failure, first) {
			if err := m.processFailure(failure); err != nil {
				log.Printf("Error reporting failure of %s: %s", failure.ID, err)
			}
		}
	}

	m.cleanupRecent()

	return nil
}

func (m *Monitor) cleanupRecent() {
	for n, ts := range m.recent {
		if time.Since(ts) > timeout {
			delete(m.recent, n)
		}
	}
}

func (m *Monitor) checkFailure(failure complainer.Failure, first bool) bool {
	if !m.recent[failure.ID].IsZero() {
		return false
	}

	m.recent[failure.ID] = failure.Finished

	if time.Since(failure.Finished) > timeout/2 {
		return false
	}

	if first {
		return false
	}

	return true
}

func (m *Monitor) processFailure(failure complainer.Failure) error {
	labels := label.NewLabels(m.name, failure.Labels, m.defaults)

	skip := true
	for n := range m.reporters {
		for range labels.Instances(n) {
			skip = false
		}
	}

	if skip {
		log.Printf("Skipping %s", failure)
		return nil
	}

	log.Printf("Reporting %s", failure)

	stdoutURL, stderrURL, err := m.mesos.Logs(failure)
	if err != nil {
		return fmt.Errorf("cannot get stdout and stderr urls from mesos: %s", err)
	}

	stdoutURL, stderrURL, err = m.uploader.Upload(failure, stdoutURL, stderrURL)
	if err != nil {
		return fmt.Errorf("cannot get stdout and stderr urls from uploader: %s", err)
	}

	for n, r := range m.reporters {
		for _, i := range labels.Instances(n) {
			config := reporter.NewConfigProvider(labels, n, i)
			if err := r.Report(failure, config, stdoutURL, stderrURL); err != nil {
				log.Printf("Cannot generate report with %s [instance=%s] for task with ID %s: %s", n, i, failure.ID, err)
			}
		}
	}

	return nil
}
