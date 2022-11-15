package config

import (
	"strings"

	"git.sr.ht/~rjarry/aerc/logging"
	"github.com/go-ini/ini"
)

type ViewerConfig struct {
	Pager          string
	Alternatives   []string
	ShowHeaders    bool       `ini:"show-headers"`
	AlwaysShowMime bool       `ini:"always-show-mime"`
	ParseHttpLinks bool       `ini:"parse-http-links"`
	HeaderLayout   [][]string `ini:"-"`
	KeyPassthrough bool       `ini:"-"`
}

func defaultViewerConfig() ViewerConfig {
	return ViewerConfig{
		Pager:        "less -R",
		Alternatives: []string{"text/plain", "text/html"},
		ShowHeaders:  false,
		HeaderLayout: [][]string{
			{"From", "To"},
			{"Cc", "Bcc"},
			{"Date"},
			{"Subject"},
		},
		ParseHttpLinks: true,
	}
}

func (config *AercConfig) parseViewer(file *ini.File) error {
	viewer, err := file.GetSection("viewer")
	if err != nil {
		goto out
	}
	if err := viewer.MapTo(&config.Viewer); err != nil {
		return err
	}
	for key, val := range viewer.KeysHash() {
		switch key {
		case "alternatives":
			config.Viewer.Alternatives = strings.Split(val, ",")
		case "header-layout":
			config.Viewer.HeaderLayout = parseLayout(val)
		}
	}
out:
	logging.Debugf("aerc.conf: [viewer] %#v", config.Viewer)
	return nil
}
