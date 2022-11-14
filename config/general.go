package config

import (
	"fmt"

	"git.sr.ht/~rjarry/aerc/logging"
	"github.com/go-ini/ini"
)

type GeneralConfig struct {
	DefaultSavePath    string `ini:"default-save-path"`
	PgpProvider        string `ini:"pgp-provider"`
	UnsafeAccountsConf bool   `ini:"unsafe-accounts-conf"`
}

func defaultGeneralConfig() GeneralConfig {
	return GeneralConfig{
		PgpProvider:        "internal",
		UnsafeAccountsConf: false,
	}
}

func (config *AercConfig) parseGeneral(file *ini.File) error {
	gen, err := file.GetSection("general")
	if err != nil {
		goto end
	}

	if err := gen.MapTo(&config.General); err != nil {
		return err
	}
	if err := config.General.validatePgpProvider(); err != nil {
		return err
	}

end:
	logging.Debugf("aerc.conf: [general] %#v", config.General)
	return nil
}

func (gen *GeneralConfig) validatePgpProvider() error {
	switch gen.PgpProvider {
	case "gpg", "internal":
		return nil
	default:
		return fmt.Errorf("pgp-provider must be either gpg or internal")
	}
}
