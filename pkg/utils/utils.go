package utils

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func Hostnames(clusterFile map[string][]string) (ret []string) {
	for _, hosts := range clusterFile {
		ret = append(ret, hosts...)
	}
	return ret
}

func CertificatePairExists(base string, dir string) bool {
	return LocalFileExists(filepath.Join(dir, fmt.Sprintf("%s.pem", base))) && LocalFileExists(filepath.Join(dir, fmt.Sprintf("%s-key.pem", base)))
}

func KeyPairExists(base string, dir string) bool {
	return LocalFileExists(filepath.Join(dir, fmt.Sprintf("%s.key", base))) && LocalFileExists(filepath.Join(dir, fmt.Sprintf("%s.pub", base)))
}

func LocalFileExists(filePath string) bool {
	if _, err := os.Stat(filePath); errors.Is(err, os.ErrNotExist) {
		return false
	}
	return true
}

func Contains(elem string, slice []string) bool {
	for _, entry := range slice {
		if entry == elem {
			return true
		}
	}

	return false
}
