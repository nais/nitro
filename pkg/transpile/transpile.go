package transpile

import (
	"encoding/json"
	"flag"
	"io"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/flatcar-linux/container-linux-config-transpiler/config"
)

func init() {
	flag.String("files-dir", "lol-hack", "")
}

func Run(src, dst, configDir string) {
	inFile, err := os.Open(src)
	if err != nil {
		log.WithError(err).Fatal("opening ignition file")
	}

	dataIn, err := io.ReadAll(inFile)
	if err != nil {
		log.WithError(err).Fatal("reading bytes")
	}

	// hack
	if err := flag.Set("files-dir", configDir); err != nil {
		log.WithError(err).Fatalf("setting files-dir flag to %s", configDir)
	}

	cfg, ast, report := config.Parse(dataIn)
	if len(report.Entries) > 0 {
		for i, entry := range report.Entries {
			log.Errorf("entry %d: %s - %s", i, entry.Kind.String(), entry.Message)
		}
		log.WithError(err).Fatalf("config parse has error entries, report: %s", report.String())
	}

	ignCfg, report := config.Convert(cfg, "", ast)
	if len(report.Entries) > 0 {
		for i, entry := range report.Entries {
			log.Errorf("entry %d: %s - %s", i, entry.Kind.String(), entry.Message)
		}
		log.WithError(err).Fatalf("config convert has error entries, report: %s", report.String())
	}

	dataOut, err := json.Marshal(&ignCfg)
	if err != nil {
		log.WithError(err).Fatalf("failed to marshal output: %v", err)
	}

	outFile, err := os.Create(dst)
	if err != nil {
		log.WithError(err).Fatalf("failed to create: %v", err)
	}

	if _, err := outFile.Write(dataOut); err != nil {
		log.WithError(err).Fatalf("failed to write: %v", err)
	}
	log.Infof("transpiled ignition file %s from %s", dst, src)
}
