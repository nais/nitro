package utils

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/nais/onprem/nitro/pkg/vars"
)

func resolveIpFake(hostname string) string {
	return "10.0.0.1"
}

func TestGenerateHosts(t *testing.T) {
	clusterWithLocation := map[string][]vars.Node{
		"cluster1": {
			{Hostname: "apiserver-ci-0", Location: "azure"},
			{Hostname: "etcd-ci-0", Location: "azure"},
			{Hostname: "etcd-ci-1", Location: "azure"},
			{Hostname: "etcd-ci-2", Location: "azure"},
			{Hostname: "worker-ci-0", Location: "azure"},
			{Hostname: "worker-ci-1", Location: "azure"},
			{Hostname: "worker-ci-4", Location: "azure"},
			{Hostname: "worker-ci-5", Location: "azure"},
			{Hostname: "prometheus-ci-0", Location: "azure"},
			{Hostname: "prometheus-ci-1", Location: "azure"},
			{Hostname: "don-not-mind-me", Location: "not-azure"},
		},
	}

	expectedResult := `10.0.0.1 apiserver-ci-0
10.0.0.1 etcd-ci-0
10.0.0.1 etcd-ci-1
10.0.0.1 etcd-ci-2
10.0.0.1 prometheus-ci-0
10.0.0.1 prometheus-ci-1
10.0.0.1 worker-ci-0
10.0.0.1 worker-ci-1
10.0.0.1 worker-ci-4
10.0.0.1 worker-ci-5
`

	result := GenerateHosts(clusterWithLocation, resolveIpFake)

	cmp := cmp.Diff(result, expectedResult)
	if cmp != "" {
		t.Errorf("GenerateHosts() mismatch (-want +got):\n%s", cmp)
	}
}
