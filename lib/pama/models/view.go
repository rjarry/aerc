package models

import (
	"bytes"
	"io"
	"strings"
	"text/template"

	"git.sr.ht/~rjarry/aerc/log"
)

var templateText = `
Project    {{.Name}}    {{if .IsActive}}[active]{{end}}
Directory  {{.Root}}
Base       {{with .Base.ID}}{{if ge (len .) 40}}{{printf "%-6.6s" .}}{{else}}{{.}}{{end}}{{end}}
{{$notes := .Notes}}{{$commits := .Commits}}
{{- range $index, $patch := .Patches}}
    {{$patch}}:
        {{- range (index $commits $patch)}}
        {{with (index $notes .ID)}}[{{.}}] {{end}}{{. -}}
        {{end}}
{{end -}}
`

var viewRenderer = template.Must(template.New("ProjectToText").Parse(templateText))

type view struct {
	Name string
	Root string
	Base Commit
	// Patches are the unique tag names.
	Patches []string
	// Commits is a map where the tag names are keys and the associated
	// commits the values.
	Commits map[string][]Commit
	// Notes contain annotations of the commits where the commit hash is
	// the key and the annotation is the value.
	Notes map[string]string
	// IsActive is true if the current project is selected.
	IsActive bool
}

func newView(p Project, active bool, notes map[string]string) view {
	v := view{
		Name:     p.Name,
		Root:     p.Root,
		Base:     p.Base,
		Commits:  make(map[string][]Commit),
		Notes:    notes,
		IsActive: active,
	}

	for _, commit := range p.Commits {
		patch := commit.Tag
		commits, ok := v.Commits[patch]
		if !ok {
			v.Patches = append(v.Patches, patch)
		}
		commits = append(commits, commit)
		v.Commits[patch] = commits
	}

	return v
}

func (v view) String() string {
	var buf bytes.Buffer
	err := viewRenderer.Execute(&buf, v)
	if err != nil {
		log.Errorf("failed to run template: %v", err)
	}
	return buf.String()
}

func (p Project) String() string {
	return newView(p, false, nil).String()
}

func (p Project) NewReader(isActive bool, notes map[string]string) io.Reader {
	return strings.NewReader(newView(p, isActive, notes).String())
}
