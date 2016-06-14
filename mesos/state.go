package mesos

type masterState struct {
	Frameworks []masterFramework `json:"frameworks"`
	Slaves     []masterSlave     `json:"slaves"`
	Pid        string            `json:"pid"`
	Leader     string            `json:"leader"`
}

type masterFramework struct {
	Name           string       `json:"name"`
	CompletedTasks []masterTask `json:"completed_tasks"`
}

type masterTask struct {
	ID        string             `json:"id"`
	Name      string             `json:"name"`
	State     string             `json:"state"`
	SlaveID   string             `json:"slave_id"`
	Labels    []masterLabel      `json:"labels"`
	Container masterContainer    `json:"container"`
	Statuses  []masterTaskStatus `json:"statuses"`
}

type masterContainer struct {
	Type   string       `json:"type"`
	Docker masterDocker `json:"docker"`
}

type masterDocker struct {
	Image string `json:"image"`
}

type masterTaskStatus struct {
	State     string  `json:"state"`
	Timestamp float64 `json:"timestamp"`
}

type masterSlave struct {
	ID   string `json:"id"`
	Host string `json:"hostname"`
}

type masterLabel struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type slaveState struct {
	CompletedFrameworks []slaveFramework `json:"completed_frameworks"`
	Frameworks          []slaveFramework `json:"frameworks"`
}

type slaveFramework struct {
	CompletedExecutors []slaveExecutor `json:"completed_executors"`
	Executors          []slaveExecutor `json:"executors"`
}

type slaveExecutor struct {
	ID        string `json:"id"`
	Directory string `json:"directory"`
}
