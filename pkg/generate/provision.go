package generate

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/nais/onprem/nitro/pkg/kubernetes"
	"github.com/nais/onprem/nitro/pkg/ssh"
	log "github.com/sirupsen/logrus"
	"github.com/sourcegraph/conc/pool"
)

func Provision(identityFile, clusterName string, nodes map[string][]string, skipDrain bool, maxConcurrency int) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	k := kubernetes.New(clusterName)
	sshClient := ssh.New("deployer", identityFile)

	wg := pool.New().WithMaxGoroutines(maxConcurrency).WithContext(ctx)
	nodeCount := maxConcurrency
	for _, role := range roleOrder() {
		for _, node := range nodes[role] {
			role, node := role, node
			if role == "worker" {
				if nodeCount > 0 {
					if nodeCount != maxConcurrency {
						time.Sleep(7 * time.Second)
					}
					nodeCount--
				}
				wg.Go(func(ctx context.Context) error {
					provision(ctx, role, node, k, sshClient, skipDrain)
					return nil
				})
			} else {
				provision(ctx, role, node, k, sshClient, skipDrain)
			}
		}
	}

	if err := wg.Wait(); err != nil {
		log.WithError(err).Error("error while waiting for workers")
	}
}

func provision(ctx context.Context, role, node string, k *kubernetes.Client, sshClient *ssh.Client, skipDrain bool) {
	start := time.Now()
	ctx = kubernetes.WithName(ctx, node)
	log := log.WithField("node", node)

	log.Infof("--- provisioning %s: %s", role, node)
	if role == "worker" && !skipDrain && !k.NewNode(ctx, node) {
		k.Drain(ctx, node)
		k.Wait(ctx, node)
		k.DeleteNode(ctx, node)
	}

	if err := sshClient.UploadFile(node, filepath.Join("output", node, "config.ign"), "/home/deployer/config.ign"); err != nil {
		log.WithError(err).Fatal("uploading ignition config")
	}

	if err := PrepareForReboot(node, sshClient); err != nil {
		log.WithError(err).Fatal("preparing reboot")
	}
	log.Info("installed new ignition config")

	log.Infof("start reboot")
	if err := sshClient.Reboot(node); err != nil {
		log.WithError(err).Info("start reboot")
	}
	if role == "etcd" {
		counter := 0
		for !EtcdHealthy(node, sshClient) {
			if counter < 5 {
				counter++
				log.Infof("etcd not healthy, sleeping for 5 seconds before rechecking")
				time.Sleep(5 * time.Second)
				continue
			}
			panic(fmt.Sprintf("etcd [%s] not healthy", node))
		}
	}

	if (role == "worker" || role == "prometheus") && !skipDrain {
		k.WaitForNode(ctx, node)
		k.LabelNode(ctx, node, "kubernetes.io/role", role)
	}
	elapsed := time.Since(start)
	log.Infof("done in %v", elapsed)
}

func roleOrder() []string {
	return []string{"etcd", "apiserver", "worker", "prometheus"}
}

func PrepareForReboot(host string, client *ssh.Client) error {
	cmd := fmt.Sprintf(`sudo mv /home/deployer/config.ign %s \
	&& sudo mkdir -p /boot/flatcar \
    && sudo touch /boot/flatcar/first_boot \
	&& sudo rm -f /etc/machine-id`, "/usr/share/oem/config.ign")

	return client.ExecuteCommand(host, cmd)
}

func EtcdHealthy(host string, client *ssh.Client) bool {
	cmd := fmt.Sprintf("/opt/etcd/bin/etcdctl endpoint health --key=/etc/ssl/etcd/etcd-client-key.pem --cacert=/etc/ssl/etcd/ca.pem --cert=/etc/ssl/etcd/etcd-client.pem --endpoints=https://%s:2379", host)

	retVal, err := client.ExecuteCommandWithOutput(host, cmd)
	if err != nil {
		log.WithError(err).Info("etcd health check failed")
	}
	log.Infof("retVal for etcdHealthy: %v", retVal)
	return strings.Contains(retVal, "is healthy")
}
