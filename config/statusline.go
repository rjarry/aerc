package config

import (
	"git.sr.ht/~rjarry/aerc/log"
	"github.com/go-ini/ini"
)

type StatuslineConfig struct {
	RenderFormat string `ini:"render-format"`
	Separator    string
	DisplayMode  string `ini:"display-mode"`
}

func defaultStatuslineConfig() StatuslineConfig {
	return StatuslineConfig{
		RenderFormat: "[%a] %S %>%T",
		Separator:    " | ",
		DisplayMode:  "",
	}
}

func (config *AercConfig) parseStatusline(file *ini.File) error {
	statusline, err := file.GetSection("statusline")
	if err != nil {
		goto out
	}
	if err := statusline.MapTo(&config.Statusline); err != nil {
		return err
	}
out:
	log.Debugf("aerc.conf: [statusline] %#v", config.Statusline)
	return nil
}
