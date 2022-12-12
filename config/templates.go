package config

import (
	"path"
	"strings"

	"git.sr.ht/~rjarry/aerc/lib/templates"
	"git.sr.ht/~rjarry/aerc/log"
	"github.com/go-ini/ini"
)

type TemplateConfig struct {
	TemplateDirs []string `ini:"template-dirs" delim:":"`
	NewMessage   string   `ini:"new-message"`
	QuotedReply  string   `ini:"quoted-reply"`
	Forwards     string   `ini:"forwards"`
}

func defaultTemplatesConfig() *TemplateConfig {
	return &TemplateConfig{
		TemplateDirs: []string{},
		NewMessage:   "new_message",
		QuotedReply:  "quoted_reply",
		Forwards:     "forward_as_body",
	}
}

var Templates = defaultTemplatesConfig()

func parseTemplates(file *ini.File) error {
	if templatesSec, err := file.GetSection("templates"); err == nil {
		if err := templatesSec.MapTo(&Templates); err != nil {
			return err
		}
		templateDirs := templatesSec.Key("template-dirs").String()
		if templateDirs != "" {
			Templates.TemplateDirs = strings.Split(templateDirs, ":")
		}
	}

	// append default paths to template-dirs
	for _, dir := range SearchDirs {
		Templates.TemplateDirs = append(
			Templates.TemplateDirs, path.Join(dir, "templates"),
		)
	}

	// we want to fail during startup if the templates are not ok
	// hence we do dummy executes here
	t := Templates
	if err := templates.CheckTemplate(t.NewMessage, t.TemplateDirs); err != nil {
		return err
	}
	if err := templates.CheckTemplate(t.QuotedReply, t.TemplateDirs); err != nil {
		return err
	}
	if err := templates.CheckTemplate(t.Forwards, t.TemplateDirs); err != nil {
		return err
	}

	log.Debugf("aerc.conf: [templates] %#v", Templates)

	return nil
}
