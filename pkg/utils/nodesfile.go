package utils

func FilterHosts(nodeConfig map[string][]string, nodes []string) map[string][]string {
	if nodes == nil {
		return nodeConfig
	}

	ret := make(map[string][]string)

	for _, n := range nodes {
		role := getRole(n, nodeConfig)
		ret[role] = append(ret[role], n)
	}

	return ret
}

func getRole(node string, nodeConfig map[string][]string) string {
	for role, nodes := range nodeConfig {
		if Contains(node, nodes) {
			return role
		}
	}
	return ""
}
