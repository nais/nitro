package templating

import (
	"io/fs"
	"os"
	"path/filepath"
	"text/template"

	log "github.com/sirupsen/logrus"
)

func templateFile(dst, src string, vars map[string]string) error {
	err := os.MkdirAll(filepath.Dir(dst), 0755)
	if err != nil {
		return err
	}

	tpl, err := template.ParseFiles(src)
	if err != nil {
		return err
	}

	output, err := os.Create(dst)
	if err != nil {
		return err
	}

	defer func(output *os.File) {
		err := output.Close()
		if err != nil {
			log.WithError(err).Fatal("close file %s", output.Name())
		}
	}(output)

	return tpl.Execute(output, vars)
}

func TemplateFiles(templateDir, outputDir string, vars map[string]string, recursive bool) {
	log.Infof("processing %s => %s", templateDir, outputDir)
	err := filepath.WalkDir(templateDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.WithError(err).Fatal("walkdir")
		}

		if path != templateDir && d.IsDir() && !recursive {
			return filepath.SkipDir
		}

		if !d.IsDir() {
			return templateFile(outputDir+"/"+d.Name(), path, vars)
		}

		return nil
	})

	if err != nil {
		log.WithError(err).Fatal("walkdir")
	}
}
