package config

import (
	"strings"

	"git.sr.ht/~rjarry/aerc/lib/log"
	"github.com/go-ini/ini"
)

type Opener struct {
	Mime string
	Args string
}

var Openers []Opener

func parseOpeners(file *ini.File) error {
	openers, err := file.GetSection("openers")
	if err != nil {
		goto out
	}

	for _, key := range openers.Keys() {
		mime := strings.ToLower(key.Name())
		Openers = append(Openers, Opener{Mime: mime, Args: key.Value()})
	}

out:
	log.Debugf("aerc.conf: [openers] %#v", Openers)
	return nil
}
