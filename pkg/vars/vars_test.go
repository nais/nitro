package vars

import (
	"reflect"
	"testing"
)

func TestForgetLocation(t *testing.T) {
	cluster := map[string][]Node{
		"group1": {
			{Hostname: "node1", Location: "location1"},
			{Hostname: "node2", Location: "location2"},
		},
		"group2": {
			{Hostname: "node3", Location: "location3"},
			{Hostname: "node4", Location: "location4"},
			{Hostname: "node5", Location: "location5"},
		},
	}

	expectedResult := map[string][]string{
		"group1": {"node1", "node2"},
		"group2": {"node3", "node4", "node5"},
	}

	result := GetHostname(cluster)

	if !reflect.DeepEqual(result, expectedResult) {
		t.Errorf("Unexpected result. Expected: %v, but got: %v", expectedResult, result)
	}
}

func TestForgetLocation_EmptyCluster(t *testing.T) {
	cluster := map[string][]Node{}

	expectedResult := map[string][]string{}

	result := GetHostname(cluster)

	if !reflect.DeepEqual(result, expectedResult) {
		t.Errorf("Unexpected result. Expected: %v, but got: %v", expectedResult, result)
	}
}
