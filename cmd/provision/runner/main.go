package main

import (
	"os"
	"path/filepath"

	"github.com/nais/onprem/nitro/pkg/generate"
	"github.com/nais/onprem/nitro/pkg/ssh"
	"github.com/nais/onprem/nitro/pkg/vars"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

type cfg struct {
	node         string
	cluster      string
	repository   string
	githubToken  string
	user         string
	identityFile string
}

func parseFlags() *cfg {
	cfg := &cfg{}

	flag.StringVar(&cfg.node, "node", "", "github runner nodes to provision")
	flag.StringVar(&cfg.cluster, "cluster", "", "kubernetes cluster")
	flag.StringVar(&cfg.repository, "repository", "", "github repository")
	flag.StringVar(&cfg.githubToken, "github-token", "", "provide github for provisioning github runners")
	flag.StringVar(&cfg.user, "user", "deployer", "user to use for ssh")
	flag.StringVar(&cfg.identityFile, "identity-file", "./id_deployer_rsa", "identity file for nodes")
	flag.Parse()

	required := []string{"node", "cluster", "repository", "github-token"}
	seen := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) { seen[f.Name] = true })
	for _, req := range required {
		if !seen[req] {
			flag.Usage()
			log.Fatalf("missing required flag %s", req)
			os.Exit(2)
		}
	}

	return cfg
}

func main() {
	flags := parseFlags()
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})

	sshClient := ssh.New(flags.user, flags.identityFile)

	nodesFile := vars.ParseSliceYAML("clusters/" + flags.cluster + ".yaml")
	apiServer := nodesFile["apiserver"][0]

	generate.RunnerConfig(flags.node, flags.cluster, apiServer, flags.identityFile, flags.githubToken, flags.repository)

	err := provision(sshClient, flags.node)
	if err != nil {
		log.Errorf("provision failed: %s", err)
		os.Exit(9)
	}
}

func provision(client *ssh.Client, runner string) error {
	log.Infof("copy ignition file to runner %s", runner)
	err := client.UploadFile(runner, filepath.Join("output", runner, "config.ign"), "/home/"+client.User()+"/config.ign")
	if err != nil {
		return err
	}

	log.Infof("prepare for provision to runner %s", runner)
	err = generate.PrepareForReboot(runner, client)
	if err != nil {
		return err
	}

	log.Infof("reboot runner %s", runner)
	err = client.Reboot(runner)
	if err != nil {
		return err
	}

	return nil
}
