package config

import (
	"errors"
	"fmt"

	"github.com/go-ini/ini"
	"github.com/google/shlex"

	"git.sr.ht/~rjarry/aerc/lib/format"
	"git.sr.ht/~rjarry/aerc/log"
	"git.sr.ht/~rjarry/aerc/models"
)

type TriggersConfig struct {
	NewEmail       string `ini:"new-email"`
	ExecuteCommand func(command []string) error
}

var Triggers = &TriggersConfig{}

func parseTriggers(file *ini.File) error {
	triggers, err := file.GetSection("triggers")
	if err != nil {
		goto out
	}
	if err := triggers.MapTo(&Triggers); err != nil {
		return err
	}
out:
	log.Debugf("aerc.conf: [triggers] %#v", Triggers)
	return nil
}

func (trig *TriggersConfig) ExecTrigger(triggerCmd string,
	triggerFmt func(string) (string, error),
) error {
	if len(triggerCmd) == 0 {
		return errors.New("Trigger command empty")
	}
	triggerCmdParts, err := shlex.Split(triggerCmd)
	if err != nil {
		return err
	}

	var command []string
	for _, part := range triggerCmdParts {
		formattedPart, err := triggerFmt(part)
		if err != nil {
			return err
		}
		command = append(command, formattedPart)
	}
	return trig.ExecuteCommand(command)
}

func (trig *TriggersConfig) ExecNewEmail(
	account *AccountConfig, msg *models.MessageInfo,
) {
	err := trig.ExecTrigger(trig.NewEmail,
		func(part string) (string, error) {
			formatstr, args, err := format.ParseMessageFormat(
				part, Ui.TimestampFormat,
				Ui.ThisDayTimeFormat,
				Ui.ThisWeekTimeFormat,
				Ui.ThisYearTimeFormat,
				format.Ctx{
					FromAddress: account.From,
					AccountName: account.Name,
					MsgInfo:     msg,
				},
			)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf(formatstr, args...), nil
		})
	if err != nil {
		log.Errorf("failed to run new-email trigger: %v", err)
	}
}
