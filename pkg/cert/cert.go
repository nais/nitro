package cert

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io"
	"io/ioutil"
	"os/exec"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

func writeCertificate(filePath string, cmd *exec.Cmd) error {
	reader, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	err = cmd.Start()
	if err != nil {
		return err
	}

	certPem := exec.Command("cfssljson", "-bare", filePath)
	writer, err := certPem.StdinPipe()
	if err != nil {
		return err
	}

	err = certPem.Start()
	if err != nil {
		return err
	}

	_, err = io.Copy(writer, reader)
	if err != nil {
		return err
	}

	err = writer.Close()
	if err != nil {
		return err
	}

	err = cmd.Wait()
	if err != nil {
		return err
	}

	err = certPem.Wait()
	if err != nil {
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
	if err := ioutil.WriteFile(filepath.Join(outputDir, name+".key"), keyPEM, 0600); err != nil {
		log.WithError(err).Fatalf("writing private key %s to file", name)
	}

	// Write public key to file.
	if err := ioutil.WriteFile(filepath.Join(outputDir, name+".pub"), pubPEM, 0644); err != nil {
		log.WithError(err).Fatalf("writing public key %s to file", name)
	}

	log.Infof("generated keypair: %s/%s.{pub,key}", outputDir, name)
}
