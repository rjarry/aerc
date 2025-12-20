package templates

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"reflect"
	"text/template"

	"git.sr.ht/~rjarry/aerc/lib/xdg"
	"git.sr.ht/~rjarry/aerc/models"
)

func findTemplate(templateName string, templateDirs []string) (string, error) {
	for _, dir := range templateDirs {
		templateFile := xdg.ExpandHome(dir, templateName)
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

// builtins is a slice of keywords and functions built into the Go standard
// library for templates. Since they are not exported, they are hardcoded here.
var builtins = []string{
	// from the Go standard library: src/text/template/parse/lex.go
	"block",
	"break",
	"continue",
	"define",
	"else",
	"end",
	"if",
	"range",
	"nil",
	"template",
	"with",

	// from the Go standard library: src/text/template/funcs.go
	"and",
	"call",
	"html",
	"index",
	"slice",
	"js",
	"len",
	"not",
	"or",
	"print",
	"printf",
	"println",
	"urlquery",
	"eq",
	"ge",
	"gt",
	"le",
	"lt",
	"ne",
}

func Terms() []string {
	var s []string
	t := reflect.TypeFor[models.TemplateData]()
	for i := 0; i < t.NumMethod(); i++ {
		s = append(s, "."+t.Method(i).Name)
	}
	for fnStr := range templateFuncs {
		s = append(s, fnStr)
	}
	s = append(s, builtins...)
	return s
}
