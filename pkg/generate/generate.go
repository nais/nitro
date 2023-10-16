package generate

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/nais/onprem/nitro/pkg/ssh"
	"github.com/nais/onprem/nitro/pkg/templating"
	"github.com/nais/onprem/nitro/pkg/transpile"
	"github.com/nais/onprem/nitro/pkg/utils"
	"github.com/nais/onprem/nitro/pkg/vars"
)

const OutputDir = "./output"

func ClusterIgnitionFiles(user, identityFile, cluster string, hosts []string) {
	err := os.RemoveAll(OutputDir)
	if err != nil {
		log.WithError(err).Fatal("deleting output dir")
	}
	log.Infof("deleted dir: %s", OutputDir)

	clusterFile := vars.ParseSliceYAML("clusters/" + cluster + ".yaml")

	variables := vars.ParseVars(cluster, identityFile, clusterFile)
	templating.TemplateFiles("templates", "output", variables, false)
	for role, roleNodes := range clusterFile {
		for _, node := range roleNodes {
			log.Infof("templating files for %s node %s\n", role, node)
			nodeDir := "output/" + node

			variables["role"] = role
			variables["hostname"] = node
			variables["hostname_short"] = strings.Split(node, ".")[0]
			variables["hostname_ip"] = vars.ResolveIP(node)

			templateDir := role
			if role == "prometheus" {
				templateDir = "worker"
			}

			templating.TemplateFiles(path.Join("templates", templateDir), nodeDir, variables, true)
		}
	}
	log.Info("finished templating")

	if hits := recursiveGrep("./output", "<no value>"); hits != nil {
		log.Errorf("found %d unresolved variables:", len(hits))
		for _, hit := range hits {
			log.Errorf(hit)
		}
		os.Exit(1)
	}
	log.Info("all variables resolved")

	log.Infof("ensuring certificates")
	filtered := utils.FilterHosts(clusterFile, hosts)
	apiServerHost := filtered["apiserver"][0]
	caDir := "output/" + apiServerHost
	sshClient := ssh.New(user, identityFile)
	ensureApiserverCerts(apiServerHost, sshClient)
	ensureKubeletCerts(merge(filtered["worker"], filtered["prometheus"]), caDir, sshClient)
	ensureEtcdCerts(filtered["etcd"], caDir, sshClient)
	log.Info("finished ensuring certificates")

	log.Info("transpiling ignition files")
	for _, host := range utils.Hostnames(filtered) {
		log.Infof("processing node %s", host)
		nodeDir := "output/" + host
		src := filepath.Join(nodeDir, "config.ign.yaml")
		dst := filepath.Join(nodeDir, "config.ign")
		transpile.Run(src, dst, nodeDir)
	}
}

func merge(slices ...[]string) (ret []string) {
	for _, s := range slices {
		ret = append(ret, s...)
	}
	return ret
}

func recursiveGrep(root string, str string) []string {
	var ret []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.WithError(err).Fatal("walking file tree")
		}
		if d != nil && !d.IsDir() {
			if hits := search(path, str); hits != nil {
				for _, hit := range hits {
					ret = append(ret, fmt.Sprintf("line %d in file %s", hit, path))
				}
			}
		}
		return nil
	})
	if err != nil {
		log.WithError(err).Fatal("walking file tree")
	}

	return ret
}

func search(file string, s string) (hits []int) {
	f, err := os.Open(file)
	if err != nil {
		log.WithError(err).Fatal("opening file")
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
		}
	}(f)

	scanner := bufio.NewScanner(f)
	lineNum := 1
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), s) {
			hits = append(hits, lineNum)
		}
		lineNum++
	}

	if err := scanner.Err(); err != nil {
		log.WithError(err).Fatal("scanning file")
	}
	return hits
}
