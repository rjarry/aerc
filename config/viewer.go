package config

import (
	"strings"

	"git.sr.ht/~rjarry/aerc/log"
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
	CloseOnReply   bool       `ini:"close-on-reply"`
}

func defaultViewerConfig() *ViewerConfig {
	return &ViewerConfig{
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
		CloseOnReply:   false,
	}
}

var Viewer = defaultViewerConfig()

func parseViewer(file *ini.File) error {
	viewer, err := file.GetSection("viewer")
	if err != nil {
		goto out
	}
	if err := viewer.MapTo(&Viewer); err != nil {
		return err
	}
	for key, val := range viewer.KeysHash() {
		switch key {
		case "alternatives":
			Viewer.Alternatives = strings.Split(val, ",")
		case "header-layout":
			Viewer.HeaderLayout = parseLayout(val)
		}
	}
out:
	log.Debugf("aerc.conf: [viewer] %#v", Viewer)
	return nil
}
