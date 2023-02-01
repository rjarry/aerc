package config

import (
	"github.com/go-ini/ini"
	"github.com/google/shlex"

	"git.sr.ht/~rjarry/aerc/lib/format"
	"git.sr.ht/~rjarry/aerc/log"
)

type TriggersConfig struct {
	NewEmail []string `ini:"-"`
}

var Triggers = &TriggersConfig{}

func parseTriggers(file *ini.File) error {
	var cmd string
	triggers, err := file.GetSection("triggers")
	if err != nil {
		goto out
	}
	if key := triggers.Key("new-email"); key != nil {
		cmd = indexFmtRegexp.ReplaceAllStringFunc(
			key.String(),
			func(s string) string {
				runes := []rune(s)
				t, _ := indexVerbToTemplate(runes[len(runes)-1])
				return t
			},
		)
		Triggers.NewEmail, err = shlex.Split(cmd)
		if err != nil {
			return err
		}
		if cmd != key.String() {
			log.Warnf("%s %s",
				"The new-email trigger now uses templates instead of %-based placeholders.",
				"Backward compatibility will be removed in aerc 0.17.")
			Warnings = append(Warnings, Warning{
				Title: "FORMAT CHANGED: [triggers].new-email",
				Body: `
The new-email trigger now uses templates instead of %-based placeholders.

Your configuration in this instance was automatically converted to:

[triggers]
new-email = ` + format.ShellQuote(Triggers.NewEmail) + `

Your configuration file was not changed. To make this change permanent and to
dismiss this warning on launch, replace the above line into aerc.conf. See
aerc-config(5) for more details.

The automatic conversion of new-email will be removed in aerc 0.17.
`,
			})
		}
	}
out:
	log.Debugf("aerc.conf: [triggers] %#v", Triggers)
	return nil
}
