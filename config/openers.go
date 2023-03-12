package config

import (
	"fmt"
	"strings"

	"git.sr.ht/~rjarry/aerc/log"
	"github.com/go-ini/ini"
	"github.com/google/shlex"
)

type Opener struct {
	Mime string
	Args []string
}

var Openers []Opener

func parseOpeners(file *ini.File) error {
	openers, err := file.GetSection("openers")
	if err != nil {
		goto out
	}

	for _, key := range openers.Keys() {
		mime := strings.ToLower(key.Name())
		if args, err := shlex.Split(key.Value()); err != nil {
			return err
		} else {
			if len(args) == 0 {
				return fmt.Errorf("opener command empty for %s", mime)
			}
			Openers = append(Openers, Opener{Mime: mime, Args: args})
		}
	}

out:
	log.Debugf("aerc.conf: [openers] %#v", Openers)
	return nil
}
