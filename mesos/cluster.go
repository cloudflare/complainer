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

// Cluster represents Mesos cluster
type Cluster struct {
	masters []string
	client  http.Client
}

// NewCluster creates a new cluster with the provided list of masters
func NewCluster(masters []string) *Cluster {
	return &Cluster{
		masters: masters,
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
			if task.State != "TASK_FAILED" && task.State != "TASK_KILLED" {
				continue
			}

			labels := map[string]string{}
			for _, label := range task.Labels {
				labels[label.Key] = label.Value
			}

			failures = append(failures, complainer.Failure{
				ID:       task.ID,
				Name:     task.Name,
				Slave:    hosts[task.SlaveID],
				State:    task.Statuses[len(task.Statuses)-1].State,
				Started:  time.Unix(int64(task.Statuses[0].Timestamp), 0),
				Finished: time.Unix(int64(task.Statuses[len(task.Statuses)-1].Timestamp), 0),
				Labels:   labels,
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
		for _, executor := range framework.CompletedExecutors {
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
