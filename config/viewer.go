package config

import (
	"sync/atomic"

	"git.sr.ht/~rjarry/aerc/lib/log"
	"github.com/go-ini/ini"
)

type ViewerConfig struct {
	Pager          string     `ini:"pager" default:"less -Rc"`
	Alternatives   []string   `ini:"alternatives" default:"text/plain,text/html" delim:","`
	ShowHeaders    bool       `ini:"show-headers"`
	AlwaysShowMime bool       `ini:"always-show-mime"`
	MaxMimeHeight  int        `ini:"max-mime-height" default:"0"`
	ParseHttpLinks bool       `ini:"parse-http-links" default:"true"`
	HeaderLayout   [][]string `ini:"header-layout" parse:"ParseLayout" default:"From|To,Cc|Bcc,Date,Subject"`
	KeyPassthrough bool
}

var viewerConfig atomic.Pointer[ViewerConfig]

func Viewer() *ViewerConfig {
	return viewerConfig.Load()
}

func parseViewer(file *ini.File) (*ViewerConfig, error) {
	conf := new(ViewerConfig)
	if err := MapToStruct(file.Section("viewer"), conf, true); err != nil {
		return nil, err
	}
	log.Debugf("aerc.conf: [viewer] %#v", conf)
	return conf, nil
}

func (v *ViewerConfig) ParseLayout(sec *ini.Section, key *ini.Key) ([][]string, error) {
	layout := parseLayout(key.String())
	return layout, nil
}
