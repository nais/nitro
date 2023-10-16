package generate

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/nais/onprem/nitro/pkg/ssh"
	"github.com/nais/onprem/nitro/pkg/templating"
	"github.com/nais/onprem/nitro/pkg/transpile"
	"github.com/nais/onprem/nitro/pkg/vars"
	log "github.com/sirupsen/logrus"
)

func RunnerConfig(node, cluster, apiServer, user, identityFile string, githubToken, repository string) {
	log.Infof("generate runner config %s\n", node)

	err := os.RemoveAll("./output")
	if err != nil {
		log.WithError(err).Fatal("deleting output dir")
	}

	variables := make(map[string]string)
	variables["apiserver"] = apiServer
	variables["cluster_name"] = cluster
	variables["github_token"] = githubToken
	variables["hostname"] = node
	variables["identity_file"] = identityFile
	variables["repository"] = repository
	variables["repository_without_slash"] = strings.ReplaceAll(repository, "/", "-")
	variables["users"] = vars.BuildUsersString(vars.ParseStringYAML("vars/admins.yaml"))

	clusterVars := vars.ParseStringYAML("vars/" + cluster + ".yaml")
	variables = vars.Merge(variables, clusterVars)

	templating.TemplateFiles("templates/github_runner", "output/"+node, variables, true)
	templating.TemplateFiles("templates", "output", variables, false)

	log.Infof("download files from API server")
	sshClient := ssh.New(user, identityFile)
	if err := sshClient.DownloadDir(apiServer, "output/"+node, "/etc/kubernetes/pki"); err != nil {
		log.Infof("could not download files from apiserver: %v", err)
	}
	nodeDir := "output/" + node
	src := filepath.Join(nodeDir, "config.ign.yaml")
	dst := filepath.Join(nodeDir, "config.ign")
	transpile.Run(src, dst, nodeDir)
}
