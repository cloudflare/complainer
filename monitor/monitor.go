package monitor

import (
	"fmt"
	"net/http"
	"net/http/pprof"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/cloudflare/complainer"
	"github.com/cloudflare/complainer/label"
	"github.com/cloudflare/complainer/matcher"
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
	matcher   matcher.FailureMatcher
	reporters map[string]reporter.Reporter
	defaults  bool
	recent    map[string]time.Time
	mu        sync.Mutex
	err       error
	log       *log.Entry
}

// NewMonitor creates the new monitor with a name, uploader and reporters
func NewMonitor(name string, cluster *mesos.Cluster, up uploader.Uploader, reporters map[string]reporter.Reporter, defaults bool, match matcher.FailureMatcher) *Monitor {
	if match == nil {
		match = &matcher.NoopMatcher{}
	}

	return &Monitor{
		name:      name,
		mesos:     cluster,
		uploader:  up,
		matcher:   match,
		reporters: reporters,
		defaults:  defaults,
		log:       log.WithField("module", "monitor"),
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
	logger := m.log.WithField("func", "handleHealthCheck")

	m.mu.Lock()
	err := m.err
	m.mu.Unlock()

	if err == nil {
		logger.Debug("GET /health: 200 OK")
		if _, httpErr := w.Write([]byte("I am mostly okay, thanks.\n")); httpErr != nil {
			logger.Errorf("/health: status healthy, but could not write HTTP response %s", httpErr)
		}

		return
	}

	logger.Warningf("GET /health: Health check failing: %s", err)
	w.WriteHeader(http.StatusInternalServerError)
	if _, httpErr := w.Write([]byte(fmt.Sprintf("Something is fishy: %s\n", err))); httpErr != nil {
		log.Errorf("/health: status unhealthy (%s), additionally could not write HTTP response: %s", err, httpErr)
	}
}

// Run does one run across failed tasks and reports any new failures
func (m *Monitor) Run() error {
	logger := m.log.WithField("func", "Run")

	logger.Debug("Retrieving failures...")
	failures, err := m.mesos.Failures()

	defer func() {
		if err == nil {
			logger.Debugf("Clearing monitor error status")
		} else {
			logger.Debugf("Setting monitor status to err: %s", err)
		}
		m.mu.Lock()
		m.err = err
		m.mu.Unlock()
	}()

	if err != nil {
		return fmt.Errorf("m.mesos.Failures(): %s", err)
	}

	first := false
	if m.recent == nil {
		m.recent = map[string]time.Time{}
		first = true
	}

	for _, failure := range failures {
		if m.checkFailure(failure, first) {
			logger.Debugf("Processing failure %s", failure.ID)
			if err := m.processFailure(failure); err != nil {
				log.Errorf("Error reporting failure of %s: %s", failure.ID, err)
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
	logger := m.log.WithFields(log.Fields{"func": "checkFailure", "failureID": failure.ID})

	if !m.matcher.Match(failure.Framework) {
		logger.Debug("Skipping (framework does not match)")
		return false
	}

	if !m.recent[failure.ID].IsZero() {
		logger.Debug("Skipping (recent)")
		return false
	}

	m.recent[failure.ID] = failure.Finished

	if time.Since(failure.Finished) > timeout/2 {
		logger.Debug("Skipping (timeout)")
		return false
	}

	if first {
		logger.Debug("Skipping (first)")
		return false
	}

	logger.Debug("Checks passed")
	return true
}

func (m *Monitor) processFailure(failure complainer.Failure) error {
	logger := m.log.WithFields(log.Fields{"func": "processFailure", "failureID": failure.ID})

	labels := label.NewLabels(m.name, failure.Labels, m.defaults)

	skip := true
	for n := range m.reporters {
		for range labels.Instances(n) {
			skip = false
		}
	}
	if skip {
		logger.Info("Skipping failure (labels)")
		return nil
	}

	logger.Info("Processing failure")

	logger.Info("Getting log URLs")
	stdoutURL, stderrURL, err := m.mesos.Logs(failure)
	if err != nil {
		return fmt.Errorf("cannot get stdout and stderr urls: m.mesos.Logs(): %s", err)
	}

	logger.Info("Uploading logs")
	stdoutURL, stderrURL, err = m.uploader.Upload(failure, stdoutURL, stderrURL)
	if err != nil {
		return fmt.Errorf("cannot get stdout and stderr urls: m.uploader.Upload(): %s", err)
	}

	logger.Info("Launching reporter(s)")
	for n, r := range m.reporters {
		for _, i := range labels.Instances(n) {
			config := reporter.NewConfigProvider(labels, n, i)
			if err := r.Report(failure, config, stdoutURL, stderrURL); err != nil {
				logger.Errorf("Cannot generate report with %s [instance=%s]: r.Report(): %s", n, i, err)
			}
		}
	}

	return nil
}
