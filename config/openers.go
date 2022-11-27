package config

import (
	"strings"

	"git.sr.ht/~rjarry/aerc/log"
	"github.com/go-ini/ini"
	"github.com/google/shlex"
)

func (config *AercConfig) parseOpeners(file *ini.File) error {
	openers, err := file.GetSection("openers")
	if err != nil {
		goto out
	}

	for mimeType, command := range openers.KeysHash() {
		mimeType = strings.ToLower(mimeType)
		if args, err := shlex.Split(command); err != nil {
			return err
		} else {
			config.Openers[mimeType] = args
		}
	}

out:
	log.Debugf("aerc.conf: [openers] %#v", config.Openers)
	return nil
}
