package config

import (
	"regexp"
	"strings"

	"git.sr.ht/~rjarry/aerc/lib/templates"
	"git.sr.ht/~rjarry/aerc/log"
	"github.com/go-ini/ini"
)

type StatuslineConfig struct {
	StatusColumns   []*ColumnDef `ini:"-"`
	ColumnSeparator string       `ini:"column-separator"`
	Separator       string       `ini:"separator"`
	DisplayMode     string       `ini:"display-mode"`
	// deprecated
	RenderFormat string `ini:"render-format"`
}

func defaultStatuslineConfig() *StatuslineConfig {
	left, _ := templates.ParseTemplate("column-left", `[{{.Account}}] {{.StatusInfo}}`)
	center, _ := templates.ParseTemplate("column-center", `{{.PendingKeys}}`)
	right, _ := templates.ParseTemplate("column-right", `{{.TrayInfo}}`)
	return &StatuslineConfig{
		StatusColumns: []*ColumnDef{
			{
				Name:     "left",
				Template: left,
				Flags:    ALIGN_LEFT | WIDTH_AUTO,
			},
			{
				Name:     "center",
				Template: center,
				Flags:    ALIGN_CENTER | WIDTH_FIT,
			},
			{
				Name:     "right",
				Template: right,
				Flags:    ALIGN_RIGHT | WIDTH_AUTO,
			},
		},
		ColumnSeparator: " ",
		Separator:       " | ",
		DisplayMode:     "text",
		// deprecated
		RenderFormat: "",
	}
}

var Statusline = defaultStatuslineConfig()

func parseStatusline(file *ini.File) error {
	statusline, err := file.GetSection("statusline")
	if err != nil {
		goto out
	}
	if err := statusline.MapTo(&Statusline); err != nil {
		return err
	}

	if key, err := statusline.GetKey("status-columns"); err == nil {
		columns, err := ParseColumnDefs(key, statusline)
		if err != nil {
			return err
		}
		Statusline.StatusColumns = columns
	} else if Statusline.RenderFormat != "" {
		columns, err := convertRenderFormat()
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
and remove index-format from it. See aerc-config(5) for more details.

index-format will be removed in aerc 0.17.
`,
		})
	}

out:
	log.Debugf("aerc.conf: [statusline] %#v", Statusline)
	return nil
}

var (
	renderFmtRe    = regexp.MustCompile(`%(-?\d+)?(\.\d+)?[acdmSTp]`)
	statuslineMute = false
)

func convertRenderFormat() ([]*ColumnDef, error) {
	var columns []*ColumnDef

	tokens := strings.Split(Statusline.RenderFormat, "%>")

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
