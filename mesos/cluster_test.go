package mesos

import (
	"reflect"
	"testing"
)

func TestNewCluster(t *testing.T) {
	inputMasters := []string{
		"http://master1.com",
		"http://master2.com/",
		"http://master3.com/fancy/path",
		"http://master4.com/fancy/path/",
	}
	expectedMasters := []string{
		"http://master1.com",
		"http://master2.com",
		"http://master3.com/fancy/path",
		"http://master4.com/fancy/path",
	}

	cluster := NewCluster(inputMasters)
	if cluster == nil {
		t.Error("Cluster is nil. Expected mesos.Cluster pointer.")
	}

	if !reflect.DeepEqual(cluster.masters, expectedMasters) {
		t.Errorf("Master list is not equal. Got %+v, expected %+v", cluster.masters, expectedMasters)
	}
}
