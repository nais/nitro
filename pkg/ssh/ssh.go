package ssh

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/melbahja/goph"
	"github.com/nais/onprem/nitro/pkg/utils"
	"github.com/nais/onprem/nitro/pkg/vars"
	log "github.com/sirupsen/logrus"
)

type Client struct {
	auth         goph.Auth
	identityFile string
	user         string
}

func New(user, privateKey string) *Client {
	auth, err := goph.Key(privateKey, "")
	if err != nil {
		log.WithError(err).Fatal("new ssh client")
	}

	return &Client{
		auth:         auth,
		identityFile: privateKey,
		user:         user,
	}
}

func (c *Client) User() string {
	return c.user
}

func (c *Client) IdentityFile() string {
	return c.identityFile
}

func (c *Client) UploadFile(host, src, dst string) error {
	ip := vars.ResolveIP(host)
	client, err := goph.NewUnknown(c.user, ip, c.auth)
	if err != nil {
		return err
	}

	return client.Upload(src, dst)
}

func (c *Client) Reboot(host string) error {
	err := c.ExecuteCommand(host, "sudo systemctl reboot")
	if err != nil {
		if "wait: remote command exited without exit status or exit signal" == err.Error() {
			return nil
		}
		return err
	}
	return nil
}

func (c *Client) downloadFile(host, dst, src string) error {
	ip := vars.ResolveIP(host)
	client, err := goph.NewUnknown(c.user, ip, c.auth)
	if err != nil {
		return err
	}

	return client.Download(src, dst)
}

func (c *Client) DownloadFile(host, dstFile, srcFile string) {
	err := c.downloadFile(host, dstFile, srcFile)
	if err != nil {
		log.Infof("could not download file %s from %s: %v", srcFile, host, err)
		if utils.LocalFileExists(dstFile) {
			f, err := os.Stat(dstFile)
			if err != nil {
				log.WithError(err).Fatalf("unable to inspect local file: %v:", dstFile)
			}
			if f.Size() == 0 {
				if err := os.Remove(dstFile); err != nil {
					log.WithError(err).Fatalf("unable to delete empty local file: %v", dstFile)
				}
			}
		}
	} else {
		log.Infof("downloaded file %s from %s", srcFile, host)
	}
}

func (c *Client) DownloadDir(host, dstDir, srcDir string) error {
	log.Infof("downloading all files from %s:%s => %s", host, srcDir, dstDir)

	out, err := c.ExecuteCommandWithOutput(host, "ls "+srcDir)
	if err != nil {
		return err
	}

	files := strings.Split(out, "\n")
	for _, file := range files {
		srcFilePath := filepath.Join(srcDir, file)
		dstFilePath := filepath.Join(dstDir, file)
		if file == "" {
			continue
		}
		c.DownloadFile(host, dstFilePath, srcFilePath)
	}

	return nil
}

func (c *Client) ExecuteCommandWithOutput(host, command string) (string, error) {
	ip := vars.ResolveIP(host)
	client, err := goph.NewUnknown(c.user, ip, c.auth)
	if err != nil {
		return "", err
	}

	defer func(client *goph.Client) {
		err := client.Close()
		if err != nil {
			log.WithError(err).Warning("close client")
		}
	}(client)

	out, err := client.Run(command)
	if err != nil {
		return "", fmt.Errorf("executing ssh ExecuteCommand: error: '%s', output: '%s'", err, string(out))
	}

	return string(out), nil
}

func (c *Client) ExecuteCommand(host, command string) error {
	ip := vars.ResolveIP(host)
	client, err := goph.NewUnknown(c.user, ip, c.auth)
	if err != nil {
		return err
	}

	client.Config.Timeout = 60 * time.Second

	out, err := client.Run(command)
	if err != nil {
		return fmt.Errorf("executing ssh command: error: '%s', output: '%s'", err, string(out))
	}

	return nil
}
