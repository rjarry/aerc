package config

import (
	"strings"

	"git.sr.ht/~rjarry/aerc/log"
	"github.com/go-ini/ini"
)

type HooksConfig struct {
	AercStartup  string `ini:"aerc-startup"`
	AercShutdown string `ini:"aerc-shutdown"`
	MailReceived string `ini:"mail-received"`
	MailDeleted  string `ini:"mail-deleted"`
}

var Hooks HooksConfig

func parseHooks(file *ini.File) error {
	err := MapToStruct(file.Section("hooks"), &Hooks, true)
	if err != nil {
		return err
	}

	newEmail := file.Section("triggers").Key("new-email").String()
	if Hooks.MailReceived == "" && newEmail != "" {
		Hooks.MailReceived = convertNewEmailTrigger(newEmail)
		Warnings = append(Warnings, Warning{
			Title: "DEPRECATION NOTICE: [triggers].new-email",
			Body: `
The new-email trigger has been replaced by [hooks].email-received.

Your configuration in this instance was automatically converted to:

[hooks]
mail-received = ` + Hooks.MailReceived + `

Please verify the accuracy of the above translation.

Your configuration file was not changed. To make this change permanent and to
dismiss this deprecation warning on launch, copy the above lines into aerc.conf
and remove new-email from it. See aerc-config(5) for more details.
`,
		})
	}

	log.Debugf("aerc.conf: [hooks] %#v", Hooks)
	return nil
}

func convertNewEmailTrigger(old string) string {
	translations := map[string]string{
		"%a": "$AERC_FROM_ADDRESS",
		"%n": "$AERC_FROM_NAME",
		"%s": "$AERC_SUBJECT",
		"%f": "$AERC_FROM_NAME <$AERC_FROM_ADDRESS>",
		"%u": `$(echo "$AERC_FROM_ADDRESS" | cut -d@ -f1)`,
		"%v": `$(echo "$AERC_FROM_NAME" | cut -d' ' -f1)`,
	}
	for replace, with := range translations {
		old = strings.ReplaceAll(old, replace, with)
	}
	old = strings.TrimPrefix(old, "exec ")
	return strings.ReplaceAll(old, "%%", "%")
}
