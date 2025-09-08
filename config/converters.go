package config

import (
	"fmt"
	"strings"
	"sync/atomic"

	"git.sr.ht/~rjarry/aerc/lib/log"
	"github.com/go-ini/ini"
)

var convertersConfig atomic.Pointer[map[string]string]

func Converters() map[string]string {
	return *convertersConfig.Load()
}

func parseConverters(file *ini.File) (map[string]string, error) {
	conf := make(map[string]string)
	converters, err := file.GetSection("multipart-converters")
	if err != nil {
		goto out
	}

	for mimeType, command := range converters.KeysHash() {
		mimeType = strings.ToLower(mimeType)
		if mimeType == "text/plain" {
			return nil, fmt.Errorf(
				"multipart-converters: text/plain is reserved")
		}
		if !strings.HasPrefix(mimeType, "text/") {
			return nil, fmt.Errorf(
				"multipart-converters: %q: only text/* MIME types are supported",
				mimeType)
		}
		conf[mimeType] = command
	}

out:
	log.Debugf("aerc.conf: [multipart-converters] %#v", conf)
	return conf, nil
}
