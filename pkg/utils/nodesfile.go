package utils

import "github.com/nais/onprem/nitro/pkg/vars"

func FilterHosts(nodeConfig map[string][]vars.Node,
	nodes []string) map[string][]vars.Node {

	if nodes == nil {
		return nodeConfig
	}

	ret := make(map[string][]vars.Node)

	for role, nodes_ := range nodeConfig {
		for _, n := range nodes_ {
			if Contains(n.Hostname, nodes) {
				ret[role] = append(ret[role], n)
			}
		}
	}
	return ret
}
