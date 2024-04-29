package utils

import (
	"sort"

	"github.com/nais/onprem/nitro/pkg/vars"
)

func GenerateHosts(clusterWithLocation map[string][]vars.Node, resolveIp func(string) string) string {
	if resolveIp == nil {
		resolveIp = vars.ResolveIP
	}

	var hostnames []string

	for _, nodes := range clusterWithLocation {
		for _, node := range nodes {
			if node.Location == "azure" {
				hostnames = append(hostnames, node.Hostname)
			}
		}
	}

	sort.Strings(hostnames)

	sortedRetVal := ""
	for _, hostname := range hostnames {
		sortedRetVal += resolveIp(hostname) + " " + hostname + "\n"
	}

	return sortedRetVal
}
