package config

import (
	"git.sr.ht/~rjarry/aerc/log"
	"github.com/go-ini/ini"
)

type HooksConfig struct {
	AercStartup  string `ini:"aerc-startup"`
	AercShutdown string `ini:"aerc-shutdown"`
	MailReceived string `ini:"mail-received"`
	MailDeleted  string `ini:"mail-deleted"`
	MailAdded    string `ini:"mail-added"`
	MailSent     string `ini:"mail-sent"`
}

var Hooks HooksConfig

func parseHooks(file *ini.File) error {
	err := MapToStruct(file.Section("hooks"), &Hooks, true)
	if err != nil {
		return err
	}

	log.Debugf("aerc.conf: [hooks] %#v", Hooks)
	return nil
}
