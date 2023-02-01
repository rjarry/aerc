package templates

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"text/template"

	"git.sr.ht/~rjarry/aerc/models"
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

func ParseTemplateFromFile(
	name string, dirs []string, data models.TemplateData,
) (io.Reader, error) {
	templateFile, err := findTemplate(name, dirs)
	if err != nil {
		return nil, err
	}
	emailTemplate, err := template.New(name).
		Funcs(templateFuncs).ParseFiles(templateFile)
	if err != nil {
		return nil, err
	}

	var body bytes.Buffer
	if err := Render(emailTemplate, &body, data); err != nil {
		return nil, err
	}
	return &body, nil
}

func ParseTemplate(name, content string) (*template.Template, error) {
	return template.New(name).Funcs(templateFuncs).Parse(content)
}

func Render(t *template.Template, w io.Writer, data models.TemplateData) error {
	return t.Execute(w, data)
}
