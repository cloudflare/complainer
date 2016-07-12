package mesos

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/cloudflare/complainer"
)

// ErrNoMesosMaster indicates that no alive mesos masters are found
var ErrNoMesosMaster = errors.New("mesos master not found")

const (
	unknownState = "UNKNOWN"
)

// Cluster represents Mesos cluster
type Cluster struct {
	masters []string
	client  http.Client
}

// NewCluster creates a new cluster with the provided list of masters
func NewCluster(masters []string) *Cluster {
	var cleanMasters []string
	for _, master := range masters {
		cleanMasters = append(cleanMasters, strings.TrimSuffix(master, "/"))
	}

	return &Cluster{
		masters: cleanMasters,
		client: http.Client{
			Timeout: time.Second * 30,
		},
	}
}

// Failures returns the list of known failes tasks
func (c *Cluster) Failures() ([]complainer.Failure, error) {
	state := &masterState{}

	for _, master := range c.masters {
		resp, err := c.client.Get(master + "/master/state")
		if err != nil {
			log.Printf("Error fetching state from %s: %s", master, err)
			continue
		}

		defer func() {
			_ = resp.Body.Close()
		}()

		err = json.NewDecoder(resp.Body).Decode(state)
		if err != nil {
			log.Printf("Error decoding state from %s: %s", master, err)
			continue
		}

		if state.Pid != state.Leader {
			continue
		}

		return c.failuresFromLeader(state), nil
	}

	return nil, ErrNoMesosMaster
}

func (c *Cluster) failuresFromLeader(state *masterState) []complainer.Failure {
	failures := []complainer.Failure{}

	hosts := map[string]string{}
	for _, slave := range state.Slaves {
		hosts[slave.ID] = slave.Host
	}

	for _, framework := range state.Frameworks {
		for _, task := range framework.CompletedTasks {
			if task.State != "TASK_FAILED" && task.State != "TASK_ERROR" && task.State != "TASK_LOST" {
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

	return failures
}

// Logs returns stdout and stderr urls fot the specified task
func (c *Cluster) Logs(failure complainer.Failure) (stdoutURL, stderrURL string, err error) {
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

				return stdoutURL, stderrURL, nil
			}
		}
	}

	return "", "", fmt.Errorf("cannot find executor by ID (%s)", failure.ID)
}

func (c *Cluster) slaveState(host string) (*slaveState, error) {
	state := &slaveState{}

	resp, err := c.client.Get("http://" + host + ":5051/state")
	if err != nil {
		return state, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

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
