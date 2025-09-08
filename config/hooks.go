package config

import (
	"sync/atomic"

	"git.sr.ht/~rjarry/aerc/lib/log"
	"github.com/go-ini/ini"
)

type HooksConfig struct {
	AercStartup  string `ini:"aerc-startup"`
	AercShutdown string `ini:"aerc-shutdown"`
	FlagChanged  string `ini:"flag-changed"`
	MailReceived string `ini:"mail-received"`
	MailDeleted  string `ini:"mail-deleted"`
	MailAdded    string `ini:"mail-added"`
	MailSent     string `ini:"mail-sent"`
	TagModified  string `ini:"tag-modified"`
}

var hooksConfig atomic.Pointer[HooksConfig]

func Hooks() *HooksConfig {
	return hooksConfig.Load()
}

func parseHooks(file *ini.File) (*HooksConfig, error) {
	conf := new(HooksConfig)
	err := MapToStruct(file.Section("hooks"), conf, true)
	if err != nil {
		return nil, err
	}

	log.Debugf("aerc.conf: [hooks] %#v", conf)
	return conf, nil
}
