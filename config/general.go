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
	PgpProvider        string       `ini:"pgp-provider"`
	UnsafeAccountsConf bool         `ini:"unsafe-accounts-conf"`
	LogFile            string       `ini:"log-file"`
	LogLevel           log.LogLevel `ini:"-"`
}

func defaultGeneralConfig() GeneralConfig {
	return GeneralConfig{
		PgpProvider:        "auto",
		UnsafeAccountsConf: false,
		LogLevel:           log.INFO,
	}
}

func (config *AercConfig) parseGeneral(file *ini.File) error {
	var level *ini.Key
	var logFile *os.File

	gen, err := file.GetSection("general")
	if err != nil {
		goto end
	}
	if err := gen.MapTo(&config.General); err != nil {
		return err
	}
	level, err = gen.GetKey("log-level")
	if err == nil {
		l, err := log.ParseLevel(level.String())
		if err != nil {
			return err
		}
		config.General.LogLevel = l
	}
	if err := config.General.validatePgpProvider(); err != nil {
		return err
	}
end:
	if !isatty.IsTerminal(os.Stdout.Fd()) {
		logFile = os.Stdout
		// redirected to file, force DEBUG level
		config.General.LogLevel = log.DEBUG
	} else if config.General.LogFile != "" {
		path, err := homedir.Expand(config.General.LogFile)
		if err != nil {
			return fmt.Errorf("log-file: %w", err)
		}
		logFile, err = os.OpenFile(path,
			os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
		if err != nil {
			return fmt.Errorf("log-file: %w", err)
		}
	}
	log.Init(logFile, config.General.LogLevel)
	log.Debugf("aerc.conf: [general] %#v", config.General)
	return nil
}

func (gen *GeneralConfig) validatePgpProvider() error {
	switch gen.PgpProvider {
	case "gpg", "internal", "auto":
		return nil
	default:
		return fmt.Errorf("pgp-provider must be either auto, gpg or internal")
	}
}
