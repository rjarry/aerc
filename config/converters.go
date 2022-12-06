package config

import (
	"fmt"
	"strings"

	"git.sr.ht/~rjarry/aerc/log"
	"github.com/go-ini/ini"
)

func (config *AercConfig) parseConverters(file *ini.File) error {
	converters, err := file.GetSection("multipart-converters")
	if err != nil {
		goto out
	}

	for mimeType, command := range converters.KeysHash() {
		mimeType = strings.ToLower(mimeType)
		if mimeType == "text/plain" {
			return fmt.Errorf(
				"multipart-converters: text/plain is reserved")
		}
		if !strings.HasPrefix(mimeType, "text/") {
			return fmt.Errorf(
				"multipart-converters: %q: only text/* MIME types are supported",
				mimeType)
		}
		config.Converters[mimeType] = command
	}

out:
	log.Debugf("aerc.conf: [multipart-converters] %#v", config.Converters)
	return nil
}
