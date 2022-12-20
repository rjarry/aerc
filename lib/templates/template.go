package templates

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"text/template"

	"github.com/mitchellh/go-homedir"
)

func findTemplate(templateName string, templateDirs []string) (string, error) {
	for _, dir := range templateDirs {
		templateFile, err := homedir.Expand(path.Join(dir, templateName))
		if err != nil {
			return "", err
		}

		if _, err := os.Stat(templateFile); os.IsNotExist(err) {
			continue
		}
		return templateFile, nil
	}

	return "", fmt.Errorf(
		"Can't find template %q in any of %v ", templateName, templateDirs)
}

func ParseTemplateFromFile(templateName string, templateDirs []string, data interface{}) (io.Reader, error) {
	templateFile, err := findTemplate(templateName, templateDirs)
	if err != nil {
		return nil, err
	}
	emailTemplate, err := template.New(templateName).
		Funcs(templateFuncs).ParseFiles(templateFile)
	if err != nil {
		return nil, err
	}

	var body bytes.Buffer
	if err := emailTemplate.Execute(&body, data); err != nil {
		return nil, err
	}
	return &body, nil
}

func CheckTemplate(templateName string, templateDirs []string) error {
	if templateName != "" {
		_, err := ParseTemplateFromFile(templateName, templateDirs, DummyData())
		return err
	}
	return nil
}
