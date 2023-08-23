package generate

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/nais/onprem/nitro/pkg/cert"
	"github.com/nais/onprem/nitro/pkg/ssh"
	"github.com/nais/onprem/nitro/pkg/utils"
	log "github.com/sirupsen/logrus"
)

func ensureKubeletCerts(hosts []string, caDir string, ssh *ssh.Client) {
	log.Info("ensuring kubelet certs")
	for _, host := range hosts {
		ensureKubeletCert(host, caDir, ssh)
	}
}

func ensureKubeletCert(hostname, caDir string, ssh *ssh.Client) {
	hostDir := fmt.Sprintf("output/%s", hostname)
	ssh.DownloadFile(hostname, filepath.Join(hostDir, "kubelet.pem"), "/etc/kubernetes/pki/kubelet.pem")
	ssh.DownloadFile(hostname, filepath.Join(hostDir, "kubelet-key.pem"), "/etc/kubernetes/pki/kubelet-key.pem")

	if !utils.CertificatePairExists("kubelet", hostDir) {
		cert.GenerateCert(hostDir+"/kubelet-csr.json", caDir, hostDir, "kubelet", "client")
	}

	log.Infof("ensured kubelet certificate for node %s", hostname)
}

func ensureEtcdCerts(hosts []string, apiServerDir string, ssh *ssh.Client) {
	for _, host := range hosts {
		workingDir := "output/" + host
		if err := ssh.DownloadDir(host, apiServerDir, "/etc/ssl/etcd/"); err != nil {
			log.Infof("could not download files from apiserver: %v", err)
		}

		shortname := strings.Split(host, ".")[0]
		if !utils.CertificatePairExists("peer-"+shortname, apiServerDir) {
			cert.GenerateCertWithConfig(workingDir+"/etcd-csr.json", workingDir+"/ca-config.json", apiServerDir+"/ca.pem", apiServerDir+"/ca-key.pem", apiServerDir, "peer-"+shortname, "peer")
		}
		if !utils.CertificatePairExists("server", apiServerDir) {
			cert.GenerateCertWithConfig(workingDir+"/etcd-csr.json", workingDir+"/ca-config.json", apiServerDir+"/ca.pem", apiServerDir+"/ca-key.pem", apiServerDir, "server", "server")
		}
		if !utils.CertificatePairExists("etcd-client", apiServerDir) {
			cert.GenerateCertWithConfig(workingDir+"/etcd-csr.json", workingDir+"/ca-config.json", apiServerDir+"/ca.pem", apiServerDir+"/ca-key.pem", apiServerDir, "etcd-client", "client")
		}
		log.Infof("ensured certs for etcd node %s", host)
	}
}

func ensureApiserverCerts(hostname string, ssh *ssh.Client) {
	log.Info("ensuring certificates for apiserver")
	workingDir := fmt.Sprintf("output/%s", hostname)
	if err := ssh.DownloadDir(hostname, workingDir, "/etc/kubernetes/pki"); err != nil {
		log.Infof("could not download files from apiserver: %v", err)
	}

	if !utils.CertificatePairExists("ca", workingDir) {
		cert.GenerateCaCert(workingDir, "output/ca-csr.json", "ca")
	}
	if !utils.KeyPairExists("sa", workingDir) {
		cert.GenerateKeyPair(workingDir, "sa", 2048)
	}
	if !utils.CertificatePairExists("front-proxy-ca", workingDir) {
		cert.GenerateCaCert(workingDir, workingDir+"/front-proxy-ca-csr.json", "front-proxy-ca")
	}
	if !utils.CertificatePairExists("front-proxy-client", workingDir) {
		cert.GenerateCertWithConfig(workingDir+"/front-proxy-client-csr.json", workingDir+"/ca-config.json", workingDir+"/front-proxy-ca.pem", workingDir+"/front-proxy-ca-key.pem", workingDir, "front-proxy-client", "client")
	}
	if !utils.CertificatePairExists("kubelet", workingDir) {
		cert.GenerateCert(workingDir+"/kubelet-csr.json", workingDir, workingDir, "kubelet", "client")
	}
	if !utils.CertificatePairExists("admin", workingDir) {
		cert.GenerateCert(workingDir+"/admin-csr.json", workingDir, workingDir, "admin", "client")
	}
	if !utils.CertificatePairExists("kube-proxy", workingDir) {
		cert.GenerateCert(workingDir+"/kube-proxy-csr.json", workingDir, workingDir, "kube-proxy", "client")
	}
	if !utils.CertificatePairExists("kube-apiserver-server", workingDir) {
		cert.GenerateCert(workingDir+"/kube-apiserver-server-csr.json", workingDir, workingDir, "kube-apiserver-server", "server")
	}

	log.Info("ensured certificates for apiserver")
}
