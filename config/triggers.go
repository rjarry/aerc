package config

import (
	"github.com/go-ini/ini"
	"github.com/google/shlex"

	"git.sr.ht/~rjarry/aerc/lib/format"
	"git.sr.ht/~rjarry/aerc/log"
)

type TriggersConfig struct {
	NewEmail []string `ini:"new-email" parse:"ParseNewEmail"`
}

var Triggers = new(TriggersConfig)

func parseTriggers(file *ini.File) error {
	if err := MapToStruct(file.Section("triggers"), Triggers, true); err != nil {
		return err
	}
	log.Debugf("aerc.conf: [triggers] %#v", Triggers)
	return nil
}

func (t *TriggersConfig) ParseNewEmail(_ *ini.Section, key *ini.Key) ([]string, error) {
	cmd := indexFmtRegexp.ReplaceAllStringFunc(
		key.String(),
		func(s string) string {
			runes := []rune(s)
			t, _ := indexVerbToTemplate(runes[len(runes)-1])
			return t
		},
	)
	args, err := shlex.Split(cmd)
	if err != nil {
		return nil, err
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
new-email = ` + format.ShellQuote(args) + `

Your configuration file was not changed. To make this change permanent and to
dismiss this warning on launch, replace the above line into aerc.conf. See
aerc-config(5) for more details.

The automatic conversion of new-email will be removed in aerc 0.17.
`,
		})
	}
	return args, nil
}
