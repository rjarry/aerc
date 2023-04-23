package config

import (
	"fmt"
	"os"

	"git.sr.ht/~rjarry/aerc/log"
	"github.com/go-ini/ini"
	"github.com/mattn/go-isatty"
	"github.com/mitchellh/go-homedir"
)

type GeneralConfig struct {
	DefaultSavePath    string       `ini:"default-save-path"`
	PgpProvider        string       `ini:"pgp-provider" default:"auto" parse:"ParsePgpProvider"`
	UnsafeAccountsConf bool         `ini:"unsafe-accounts-conf"`
	LogFile            string       `ini:"log-file"`
	LogLevel           log.LogLevel `ini:"log-level" default:"info" parse:"ParseLogLevel"`
	DisableIPC         bool         `ini:"disable-ipc"`
	EnableOSC8         bool         `ini:"enable-osc8" default:"false"`
	Term               string       `ini:"term" default:"xterm-256color"`
}

var General = new(GeneralConfig)

func parseGeneral(file *ini.File) error {
	var logFile *os.File

	if err := MapToStruct(file.Section("general"), General, true); err != nil {
		return err
	}
	if !isatty.IsTerminal(os.Stdout.Fd()) {
		logFile = os.Stdout
		// redirected to file, force TRACE level
		General.LogLevel = log.TRACE
	} else if General.LogFile != "" {
		path, err := homedir.Expand(General.LogFile)
		if err != nil {
			return fmt.Errorf("log-file: %w", err)
		}
		logFile, err = os.OpenFile(path,
			os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
		if err != nil {
			return fmt.Errorf("log-file: %w", err)
		}
	}
	log.Init(logFile, General.LogLevel)
	log.Debugf("aerc.conf: [general] %#v", General)
	return nil
}

func (gen *GeneralConfig) ParseLogLevel(sec *ini.Section, key *ini.Key) (log.LogLevel, error) {
	return log.ParseLevel(key.String())
}

func (gen *GeneralConfig) ParsePgpProvider(sec *ini.Section, key *ini.Key) (string, error) {
	switch key.String() {
	case "gpg", "internal", "auto":
		return key.String(), nil
	}
	return "", fmt.Errorf("must be either auto, gpg or internal")
}
