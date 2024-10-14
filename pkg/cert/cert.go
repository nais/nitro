package cert

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

func writeCertificate(filePath string, cmd *exec.Cmd) error {
	reader, err := cmd.StdoutPipe()
	if err != nil {
		log.Infof("error creating stdout pipe: %s", err)
		return err
	}

	err = cmd.Start()
	if err != nil {
		log.Infof("error starting command: %s", err)
		return err
	}

	certPem := exec.Command("cfssljson", "-bare", filePath)
	writer, err := certPem.StdinPipe()
	if err != nil {
		log.Infof("error creating stdin pipe: %s", err)
		return err
	}

	err = certPem.Start()
	if err != nil {
		log.Infof("error starting command: %s", err)
		return err
	}

	_, err = io.Copy(writer, reader)
	if err != nil {
		log.Infof("error copying data: %s", err)
		return err
	}

	err = writer.Close()
	if err != nil {
		log.Infof("error closing writer: %s", err)
		return err
	}

	err = cmd.Wait()
	if err != nil {
		log.Infof("error waiting for command: %s", err)
		return err
	}

	err = certPem.Wait()
	if err != nil {
		log.Infof("error waiting for command: %s", err)
		return err
	}
	return nil
}

func GenerateCaCert(outputDir, csrPath, name string) {
	cmd := exec.Command("cfssl", "gencert", "-initca", csrPath)

	err := writeCertificate(filepath.Join(outputDir, name), cmd)
	if err != nil {
		log.WithError(err).Fatalf("generating ca certificate from %s", csrPath)
	}
	log.Infof("generated CA cert: %s/%s{,-key}.pem", outputDir, name)
}

func GenerateCertWithConfig(csrPath, caConfig, caPublic, caKey, outputDir, name, profile string) {
	cmd := exec.Command("cfssl",
		"gencert",
		"-ca="+caPublic,
		"-ca-key="+caKey,
		"-config="+caConfig,
		"-profile="+profile,
		csrPath)

	err := writeCertificate(outputDir+"/"+name, cmd)
	if err != nil {
		log.WithError(err).Fatalf("generating %s certificate: %s from csr: %s using CA file pair [%s:%s] with CA-config %s", profile, outputDir+"/"+name+"{,-key.pem}", csrPath, caPublic, caKey, caConfig)
	}
	log.Infof("generated cert: %s/%s{,-key}.pem (%s)", outputDir, name, profile)
}
func GenerateCert(csrPath, caDir, outputDir, name, profile string) {
	GenerateCertWithConfig(csrPath, caDir+"/ca-config.json", caDir+"/ca.pem", caDir+"/ca-key.pem", outputDir, name, profile)
}

func GenerateKeyPair(outputDir, name string, bitSize int) {
	key, err := rsa.GenerateKey(rand.Reader, bitSize)
	if err != nil {
		log.WithError(err).Fatalf("generating private key %s", name)
	}

	keyPEM := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(key),
		},
	)

	pubBytes, err := x509.MarshalPKIXPublicKey(key.Public())
	if err != nil {
		log.WithError(err).Fatalf("generating public key %s", name)
	}

	pubPEM := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PUBLIC KEY",
			Bytes: pubBytes,
		},
	)

	// Write private key to file.
	if err := os.WriteFile(filepath.Join(outputDir, name+".key"), keyPEM, 0600); err != nil {
		log.WithError(err).Fatalf("writing private key %s to file", name)
	}

	// Write public key to file.
	if err := os.WriteFile(filepath.Join(outputDir, name+".pub"), pubPEM, 0644); err != nil {
		log.WithError(err).Fatalf("writing public key %s to file", name)
	}

	log.Infof("generated keypair: %s/%s.{pub,key}", outputDir, name)
}

func GetSubjectAlternativeNames(certName string) ([]string, []net.IP, error) {
	certFile, err := os.ReadFile(certName)
	if err != nil {
		return nil, nil, fmt.Errorf("error reading certificate file: %s", err)
	}

	// Decode the PEM encoded certificate
	block, _ := pem.Decode(certFile)
	if block == nil {
		fmt.Println("Error decoding certificate PEM")
		return nil, nil, fmt.Errorf("error decoding certificate PEM: %s", err)
	}

	// Parse the certificate
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("error parsing certificate: %s", err)
	}

	return cert.DNSNames, cert.IPAddresses, nil
}
