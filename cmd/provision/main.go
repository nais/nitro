package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/nais/onprem/nitro/pkg/analyze"
	"github.com/nais/onprem/nitro/pkg/generate"
	"github.com/nais/onprem/nitro/pkg/ssh"
	"github.com/nais/onprem/nitro/pkg/utils"
	"github.com/nais/onprem/nitro/pkg/vars"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	_ "k8s.io/client-go/plugin/pkg/client/auth/azure"
)

var cfg struct {
	cluster        string
	identityFile   string
	user           string
	hosts          []string
	skipDrain      bool
	maxParallelism int
}

func getSupportedCommands() []string {
	return []string{"generate", "provision", "analyze"}
}

func init() {
	flag.StringSliceVar(&cfg.hosts, "hosts", nil, "limit provisioning to specific hosts (delimiter ',').")
	flag.StringVar(&cfg.cluster, "cluster", "", "which cluster to perform actions")
	flag.StringVar(&cfg.identityFile, "identity-file", "./id_deployer_rsa", "identity file for nodes")
	flag.StringVar(&cfg.user, "user", "deployer", "user to use for ssh")
	flag.BoolVar(&cfg.skipDrain, "skipDrain", false, "run without setting NoExecute taint and NoSchedule on nodes")
	flag.IntVar(&cfg.maxParallelism, "maxParallelism", 2, "max number of parallel nodes for provisioning")
}

func main() {
	flag.Parse()
	if len(cfg.cluster) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	command := os.Args[1]
	if !utils.Contains(command, getSupportedCommands()) {
		log.Fatalf("argument must be one of: %s", getSupportedCommands())
		os.Exit(1)
	}

	setupLogging()

	log.Infof("nais ignition template resolver [operation: %s, cluster: %s]", command, cfg.cluster)

	sshClient := ssh.New(cfg.user, cfg.identityFile)

	if command == "generate" {
		generate.ClusterIgnitionFiles(sshClient, cfg.cluster, cfg.hosts)
	}

	if command == "analyze" {
		clusterFile := vars.ParseSliceYAML("clusters/" + cfg.cluster + ".yaml")
		roleHosts := calculateHosts(clusterFile, sshClient, "output")
		changes := []string{fmt.Sprintf("## %s\n", cfg.cluster)}
		for role, hosts := range roleHosts {
			for _, host := range hosts {
				diff := analyze.Analyze(sshClient, host)
				changes = append(changes, fmt.Sprintf("### %s - %s\n%s\n", role, host, diff))
			}
		}
		diffBytes := []byte(strings.Join(changes, ""))
		err := os.WriteFile("output/analysis.out", diffBytes, 0o644)
		if err != nil {
			log.WithError(err).Fatal("write analysis.out")
		}
	}

	if command == "provision" {
		clusterFile := vars.ParseSliceYAML("clusters/" + cfg.cluster + ".yaml")
		hosts := calculateHosts(clusterFile, sshClient, "output")
		if hosts == nil {
			log.Infof("no hosts to provision. exiting")
			os.Exit(0)
		}

		generate.Provision(sshClient, cfg.cluster, hosts, cfg.skipDrain, cfg.maxParallelism)
	}
}

func calculateHosts(clusterFile map[string][]string, sshClient *ssh.Client, outputDir string) map[string][]string {
	log.Infof("checking which nodes has changes...")
	if cfg.hosts != nil {
		return utils.FilterHosts(clusterFile, cfg.hosts)
	}

	var nodes []string
	for _, host := range utils.Hostnames(clusterFile) {
		remoteSum := ""
		localSum := sha256sum(outputDir + "/" + host + "/config.ign")

		ok, _ := sshClient.ExecuteCommandWithOutput(host, "sudo test -e  /usr/share/oem/config.ign && echo exists")
		log.Infof("host %s@%s: %s", sshClient.User(), host, ok)
		if ok == "exists" {
			current, err := sshClient.ExecuteCommandWithOutput(host, "sudo sha256sum /usr/share/oem/config.ign")
			if err != nil {
				log.WithError(err).Fatalf("getting checksum of current ignition file for host %s@%s", sshClient.User(), host)
			}
			remoteSum = strings.Split(current, " ")[0]
		}

		if remoteSum != localSum {
			nodes = append(nodes, host)
		}
	}
	if len(nodes) == 0 {
		return nil
	}

	return utils.FilterHosts(clusterFile, nodes)
}

func sha256sum(path string) string {
	f, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			log.WithError(err).Fatal("sha256sum")
		}
	}(f)

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		log.Fatal(err)
	}

	return hex.EncodeToString(h.Sum(nil))
}

func setupLogging() {
	file, err := os.OpenFile("nitro.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o666)
	if err != nil {
		fmt.Println("Could Not Open Log File : " + err.Error())
	}
	log.SetOutput(io.MultiWriter(file, os.Stdout))
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
}
