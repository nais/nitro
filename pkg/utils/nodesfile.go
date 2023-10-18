package utils

import "github.com/nais/onprem/nitro/pkg/vars"

func FilterHosts(nodeConfig map[string][]vars.Node,
	nodes []string) map[string][]vars.Node {
	if nodes == nil {
		return nodeConfig
	}

	ret := make(map[string][]vars.Node)

	for _, n := range nodes {
		role := getRole(n, nodeConfig)
		ret[role] = append(ret[role], nodeConfig[role])
	}
	return ret
}

// Role as in Etcd, Apiserver, worker, etc
func getRole(node string, nodeConfig map[string][]vars.Node) string {
	for role, nodes := range vars.ForgetLocation(nodeConfig) {
		if Contains(node, nodes) {
			return role
		}
	}
	return ""
}
