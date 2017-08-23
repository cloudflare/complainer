package mesos

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/cloudflare/complainer"
)

// ErrNoMesosMaster indicates that no alive mesos masters are found
var ErrNoMesosMaster = errors.New("mesos master not found")

const (
	unknownState = "UNKNOWN"
)

// Cluster represents Mesos cluster
type Cluster struct {
	masters     []string
	client      http.Client
	log         *log.Entry
	logAllTasks bool
}

// NewCluster creates a new cluster with the provided list of masters
func NewCluster(masters []string, logAllTasks bool) *Cluster {
	var cleanMasters []string
	for _, master := range masters {
		cleanMasters = append(cleanMasters, strings.TrimSuffix(master, "/"))
	}

	logger := log.WithField("module", "mesos")
	logger.Debugf("Masters: %+v", cleanMasters)

	return &Cluster{
		masters: cleanMasters,
		client: http.Client{
			Timeout: time.Second * 30,
		},
		log:         logger,
		logAllTasks: logAllTasks,
	}
}

// Failures returns the list of known failed tasks
func (c *Cluster) Failures() ([]complainer.Failure, error) {
	state := &masterState{}
	logger := c.log.WithField("func", "Failures")

	for _, master := range c.masters {
		logger.Debug("Getting state from " + master + "/master/state")
		resp, err := c.client.Get(master + "/master/state")
		if err != nil {
			logger.Errorf("Error fetching state from %s: %s", master, err)
			continue
		}

		defer func() {
			err := resp.Body.Close()
			if err != nil {
				logger.Warnf("Failed to close response body: %s", err)
			}
		}()

		logger.Debugf("Decoding master state...")
		err = json.NewDecoder(resp.Body).Decode(state)
		if err != nil {
			logger.Errorf("Error decoding JSON state data from %s: %s", master, err)
			continue
		}

		if state.Pid != state.Leader {
			logger.Debugf("Master %s is not the leader: state.Pid %s != state.Leader %s",
				master, state.Pid, state.Leader)
			continue
		}

		logger.Debugf("Leader found, retrieving failures: %s", master)
		return c.failuresFromLeader(state), nil
	}

	logger.Error("No leading master found!")
	return nil, ErrNoMesosMaster
}

func (c *Cluster) failuresFromLeader(state *masterState) []complainer.Failure {
	logger := c.log.WithField("func", "failuresFromLeader")
	failures := []complainer.Failure{}

	hosts := map[string]string{}
	for _, slave := range state.Slaves {
		hosts[slave.ID] = slave.Host
	}

	totalCompleted := 0
	totalOk := 0
	for _, framework := range state.Frameworks {
		logger.Debugf("Framework %s: scanning %d completed tasks", framework.Name, len(framework.CompletedTasks))
		totalCompleted += len(framework.CompletedTasks)

		for _, task := range framework.CompletedTasks {
			if task.State != "TASK_FAILED" && task.State != "TASK_ERROR" && task.State != "TASK_LOST" {
				// This would be too verbose even for normal debugging (1000 completed tasks/framework)
				if c.logAllTasks {
					logger.Debugf("Task OK: %s: %s/%s on %s",
						framework.Name, task.Name, task.ID, hosts[task.SlaveID])
				}
				totalOk++
				continue
			}

			labels := map[string]string{}
			for _, label := range task.Labels {
				labels[label.Key] = label.Value
			}

			// The following is to handle the case where mesos tasks don't have any statuses
			var startedAt int64
			var finishedAt int64
			var state = unknownState

			if len(task.Statuses) > 0 {
				startedAt = int64(task.Statuses[0].Timestamp)
				finishedAt = int64(task.Statuses[len(task.Statuses)-1].Timestamp)
				state = task.Statuses[len(task.Statuses)-1].State
			}

			if c.logAllTasks {
				logger.Debugf("Task FAILED: %s: %s/%s: on %s/%s, t0=%s, t1=%s, %+v",
					framework.Name, task.Name, task.ID, hosts[task.SlaveID], task.SlaveID,
					time.Unix(startedAt, 0), time.Unix(finishedAt, 0), labels)
			}

			failures = append(failures, complainer.Failure{
				ID:        task.ID,
				Name:      task.Name,
				Slave:     hosts[task.SlaveID],
				Framework: framework.Name,
				Image:     task.Container.Docker.Image,
				State:     state,
				Started:   time.Unix(startedAt, 0),
				Finished:  time.Unix(finishedAt, 0),
				Labels:    labels,
			})
		}
	}

	logger.Debugf("Found %d failed and %d successful tasks, %d in total", len(failures), totalOk, totalCompleted)
	return failures
}

// Logs returns stdout and stderr urls fot the specified task
func (c *Cluster) Logs(failure complainer.Failure) (stdoutURL, stderrURL string, err error) {
	logger := c.log.WithField("func", "Logs")
	state, err := c.slaveState(failure.Slave)
	if err != nil {
		return "", "", err
	}

	for _, framework := range append(state.Frameworks, state.CompletedFrameworks...) {
		// Tasks are not necessarily promoted to completed immediately,
		// that's why we need to look at current executors too.
		for _, executor := range append(framework.Executors, framework.CompletedExecutors...) {
			if executor.ID == failure.ID {
				stdoutURL = sandboxURL(failure.Slave, executor.Directory, "stdout")
				stderrURL = sandboxURL(failure.Slave, executor.Directory, "stderr")

				logger.Debugf("Sandbox stdout URL for %s: %s", failure.ID, stdoutURL)
				logger.Debugf("Sandbox stderr URL for %s: %s", failure.ID, stderrURL)
				return stdoutURL, stderrURL, nil
			}
		}
	}

	return "", "", fmt.Errorf("No executor found while retrievind sandbox stdout/stderr URLs for ID (%s)", failure.ID)
}

func (c *Cluster) slaveState(host string) (*slaveState, error) {
	logger := c.log.WithField("func", "slaveState")
	state := &slaveState{}
	url := "http://" + host + ":5051/state"

	logger.Debugf("Retrieving slave state from %s", url)
	resp, err := c.client.Get(url)
	if err != nil {
		return state, fmt.Errorf("Could not retrieve slave state from %s: %s", url, err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	logger.Debugf("Decoding slave state...")
	return state, json.NewDecoder(resp.Body).Decode(state)
}

func sandboxURL(host, directory, file string) string {
	return (&url.URL{
		Scheme:   "http",
		Host:     host + ":5051",
		Path:     "files/download",
		RawQuery: "path=" + directory + "/" + file,
	}).String()
}
