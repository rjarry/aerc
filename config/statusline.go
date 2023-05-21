package config

import (
	"regexp"
	"strings"

	"git.sr.ht/~rjarry/aerc/lib/templates"
	"git.sr.ht/~rjarry/aerc/log"
	"github.com/go-ini/ini"
)

type StatuslineConfig struct {
	StatusColumns   []*ColumnDef `ini:"status-columns" parse:"ParseColumns" default:"left<*,center>=,right>*"`
	ColumnSeparator string       `ini:"column-separator" default:" "`
	Separator       string       `ini:"separator" default:" | "`
	DisplayMode     string       `ini:"display-mode" default:"text"`
}

var Statusline = new(StatuslineConfig)

func parseStatusline(file *ini.File) error {
	statusline := file.Section("statusline")
	if err := MapToStruct(statusline, Statusline, true); err != nil {
		return err
	}

	if key, err := statusline.GetKey("render-format"); err == nil {
		columns, err := convertRenderFormat(key.String())
		if err != nil {
			return err
		}
		Statusline.StatusColumns = columns
		log.Warnf("%s %s",
			"The [statusline] render-format setting has been replaced by status-columns.",
			"render-format will be removed in aerc 0.17.")
		Warnings = append(Warnings, Warning{
			Title: "DEPRECATION WARNING: [statusline].render-format",
			Body: `
The render-format setting is deprecated. It has been replaced by status-columns.

Your configuration in this instance was automatically converted to:

[statusline]
` + ColumnDefsToIni(columns, "status-columns") + `
Your configuration file was not changed. To make this change permanent and to
dismiss this deprecation warning on launch, copy the above lines into aerc.conf
and remove render-format from it. See aerc-config(5) for more details.

render-format will be removed in aerc 0.17.
`,
		})
	}

	log.Debugf("aerc.conf: [statusline] %#v", Statusline)
	return nil
}

func (s *StatuslineConfig) ParseColumns(sec *ini.Section, key *ini.Key) ([]*ColumnDef, error) {
	if !sec.HasKey("column-left") {
		_, _ = sec.NewKey("column-left", "[{{.Account}}] {{.StatusInfo}}")
	}
	if !sec.HasKey("column-center") {
		_, _ = sec.NewKey("column-center", "{{.PendingKeys}}")
	}
	if !sec.HasKey("column-right") {
		_, _ = sec.NewKey("column-right", "{{.TrayInfo}}")
	}
	return ParseColumnDefs(key, sec)
}

var (
	renderFmtRe    = regexp.MustCompile(`%(-?\d+)?(\.\d+)?[acdmSTp]`)
	statuslineMute = false
)

func convertRenderFormat(renderFormat string) ([]*ColumnDef, error) {
	var columns []*ColumnDef

	tokens := strings.Split(renderFormat, "%>")

	left := renderFmtRe.ReplaceAllStringFunc(
		tokens[0], renderVerbToTemplate)
	left = strings.TrimSpace(left)
	t, err := templates.ParseTemplate("column-left", left)
	if err != nil {
		return nil, err
	}
	columns = append(columns, &ColumnDef{
		Name:     "left",
		Template: t,
		Flags:    ALIGN_LEFT | WIDTH_AUTO,
	})

	if len(tokens) == 2 {
		right := renderFmtRe.ReplaceAllStringFunc(
			tokens[1], renderVerbToTemplate)
		right = strings.TrimSpace(right)
		t, err := templates.ParseTemplate("column-right", right)
		if err != nil {
			return nil, err
		}
		columns = append(columns, &ColumnDef{
			Name:     "right",
			Template: t,
			Flags:    ALIGN_RIGHT | WIDTH_AUTO,
		})
	}

	if statuslineMute {
		columns = nil
	}

	return columns, nil
}

func renderVerbToTemplate(verb string) (template string) {
	switch verb[len(verb)-1] {
	case 'a':
		template = `{{.Account}}`
	case 'c':
		template = `{{.ConnectionInfo}}`
	case 'd':
		template = `{{.Folder}}`
	case 'S':
		template = `{{.StatusInfo}}`
	case 'T':
		template = `{{.TrayInfo}}`
	case 'p':
		template = `{{cwd}}`
	case 'm':
		statuslineMute = true
	}
	return template
}
