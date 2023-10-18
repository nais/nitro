package vars

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"sort"
	"strings"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type Node struct {
	Hostname string `yaml:"name"`
	Location string `yaml:"location"`
}

func ForgetLocation(cluster map[string][]Node) map[string][]string {
	result := make(map[string][]string)

	for key, nodes := range cluster {
		hostnames := make([]string, len(nodes))
		for i, node := range nodes {
			hostnames[i] = node.Hostname
		}
		result[key] = hostnames
	}

	return result
}

func ParseVars(cluster, identity string, hosts map[string][]Node) map[string]string {
	vars := ParseStringYAML("vars/" + cluster + ".yaml") // read cluster-specific vars
	vars["users"] = BuildUsersString(ParseStringYAML("vars/admins.yaml"))
	vars["identity_file"] = identity
	vars["cluster_name"] = cluster

	additionalVars := resolveRuntimeVars(hosts)

	return Merge(vars, additionalVars)
}

func resolveRuntimeVars(hosts map[string][]Node) map[string]string {
	nodes := make(map[string][]string)
	for _, node := range hosts["worker"] {
		nodes["worker"] = append(nodes["worker"], node.Hostname)
	}
	for _, node := range hosts["etcd"] {
		nodes["etcd"] = append(nodes["etcd"], node.Hostname)
	}
	noProxyIPs := resolveIPs(append(nodes["worker"], nodes["prometheus"]...))

	var etcdIPList []string
	var etcdUrls []string
	var etcdInitialCluster []string
	for _, etcdUrl := range nodes["etcd"] {
		etcdUrls = append(etcdUrls, "https://"+etcdUrl+":2379")
		etcdServerShort := strings.Split(etcdUrl, ".")[0]
		ip := ResolveIP(etcdUrl)

		etcdIPList = append(etcdIPList, ip)
		shortString := fmt.Sprintf("%s=https://%s:2380", etcdServerShort, ip)
		etcdInitialCluster = append(etcdInitialCluster, shortString)
	}

	vars := make(map[string]string)
	vars["apiserver"] = nodes["apiserver"][0]
	vars["apiserver_ip"] = ResolveIP(nodes["apiserver"][0])
	vars["worker_ips"] = strings.Join(noProxyIPs, ",")
	vars["etcd_hostnames"] = strings.Join(nodes["etcd"], "\",\n\"")
	vars["etcd_ips"] = strings.Join(etcdIPList, "\",\n\"")
	vars["etcd_initial_cluster"] = strings.Join(etcdInitialCluster, ",")
	vars["etcd_urls"] = strings.Join(etcdUrls, ",")

	log.Infof("resolved runtime vars")
	return vars
}

func ParseStringYAML(file string) map[string]string {
	f, err := os.ReadFile(file)
	if err != nil {
		log.WithError(err).Fatalf("reading yaml file: %s", file)
	}

	vars := make(map[string]string)
	err = yaml.Unmarshal(f, &vars)
	if err != nil {
		log.WithError(err).Fatalf("unmarshalling yaml file: %s", file)
	}

	return vars
}

func ParseClusterYAML(file string) map[string][]Node {
	f, err := os.ReadFile(file)
	if err != nil {
		log.WithError(err).Fatalf("reading yaml file: %s", file)
	}

	vars := make(map[string][]Node)
	err = yaml.Unmarshal(f, &vars)
	if err != nil {
		log.WithError(err).Fatalf("unmarshalling yaml file: %s", file)
	}

	return vars
}

func ParseSliceYAML(file string) map[string][]string {
	f, err := os.ReadFile(file)
	if err != nil {
		log.WithError(err).Fatalf("reading yaml file: %s", file)
	}

	vars := make(map[string][]string)
	err = yaml.Unmarshal(f, &vars)
	if err != nil {
		log.WithError(err).Fatalf("unmarshalling yaml file: %s", file)
	}

	return vars
}

func resolveIPs(hostnames []string) []string {
	var ips []string
	for _, hostname := range hostnames {
		ip := ResolveIP(hostname)
		ips = append(ips, ip)
	}
	return ips
}

func ResolveIP(hostname string) string {
	// Resolves DNS through aura jumphost
	if os.Getenv("JUMPHOST_DNS") != "" {
		out, err := exec.Command("/usr/bin/ssh", "aura", "dig +short "+hostname).Output()
		if err != nil {
			log.WithError(err).Fatalf("resolving ip for %s", hostname)
		}
		return strings.TrimSuffix(string(out), "\n")
	}

	ip, err := net.LookupIP(hostname)
	if err != nil {
		log.WithError(err).Fatalf("resolving ip for %s", hostname)
	}
	return ip[0].String()
}

func BuildUsersString(users map[string]string) string {
	var keys []string
	for key := range users {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var b strings.Builder

	for _, key := range keys {
		b.WriteString(fmt.Sprintf("    - name: %s\n", key))
		b.WriteString("      groups: [sudo]\n")
		b.WriteString("      ssh_authorized_keys:\n")
		b.WriteString(fmt.Sprintf("      - \"%s\"\n", users[key]))
	}

	return b.String()
}

func Merge(base, override map[string]string) map[string]string {
	ret := make(map[string]string)

	for k, v := range base {
		ret[k] = v
	}

	for k, v := range override {
		ret[k] = v
	}

	return ret
}
