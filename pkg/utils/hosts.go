package utils

import "github.com/nais/onprem/nitro/pkg/vars"

func GenerateHosts(clusterWithLocation map[string][]vars.Node) string {
	retVal := ""
	for _, nodes := range clusterWithLocation {
		for _, node := range nodes {
			if node.Location == "azure" {
				ip := vars.ResolveIP(node.Hostname)
				retVal += ip + " " + node.Hostname + "\n"
			}
		}
	}
	return retVal
}
