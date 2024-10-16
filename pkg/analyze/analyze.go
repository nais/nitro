package analyze

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/flatcar/ignition/config/v2_3/types"
	"github.com/nais/onprem/nitro/pkg/ssh"
	"github.com/r3labs/diff/v2"
	log "github.com/sirupsen/logrus"
)

const (
	remoteIgnitionFile = "config.ign.remote.yaml"
)

func Analyze(sshClient *ssh.Client, host string) string {
	localIgnitionFile := readFileBytes(fmt.Sprintf("output/%s/config.ign", host))

	sshClient.DownloadFile(host, path.Join("output", host, remoteIgnitionFile), "/usr/share/oem/config.ign")

	remoteIgnitionFile := readFileBytes(fmt.Sprintf("output/%s/%s", host, remoteIgnitionFile))

	var localIgnitionConfig types.Config
	err := json.Unmarshal(localIgnitionFile, &localIgnitionConfig)
	if err != nil {
		log.WithError(err).Fatal("unmarshal local ignition file")
	}
	var remoteIgnitionConfig types.Config
	err = json.Unmarshal(remoteIgnitionFile, &remoteIgnitionConfig)
	if err != nil {
		log.WithError(err).Fatal("unmarshal remote ignition file")
	}

	differ, err := diff.NewDiffer(diff.TagName("json"))
	if err != nil {
		log.WithError(err).Fatal("new differ")
	}
	changelog, err := differ.Diff(remoteIgnitionConfig, localIgnitionConfig)
	if err != nil {
		return ""
	}
	stringBuilder := buildMarkdownTable(changelog)
	return stringBuilder
}

func buildMarkdownTable(changelog diff.Changelog) string {
	var stringBuilder strings.Builder
	stringBuilder.WriteString("| Type | Path | Change |\n")
	stringBuilder.WriteString("| :---: | :--- | :--- |\n")

	for _, change := range changelog {
		switch change.Type {
		case diff.DELETE:
			stringBuilder.WriteString(fmt.Sprintf("| %s | %s | %s |\n", change.Type, strings.Join(change.Path, "."), change.From))
		case diff.CREATE:
			stringBuilder.WriteString(fmt.Sprintf("| %s | %s | %s |\n", change.Type, strings.Join(change.Path, "."), change.To))
		default:
			stringBuilder.WriteString(fmt.Sprintf("| %s | %s | %s |\n| | | %s |\n", change.Type, strings.Join(change.Path, "."), change.From, change.To))
		}
	}
	return stringBuilder.String()
}

func readFileBytes(filePath string) []byte {
	file, err := os.Open(filePath)
	if err != nil {
		log.WithError(err).Fatal("open file")
	}

	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.WithError(err).Fatal("close file")
		}
	}(file)

	byteSlice, err := io.ReadAll(file)
	if err != nil {
		log.WithError(err).Fatal("read file")
	}
	return byteSlice
}
