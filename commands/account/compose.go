package account

import (
	"errors"

	"git.sr.ht/~sircmpwn/aerc/widgets"
)

type Compose struct{}

func init() {
	register(Compose{})
}

func (_ Compose) Aliases() []string {
	return []string{"compose"}
}

func (_ Compose) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

// TODO: Accept arguments for default headers, message body
func (_ Compose) Execute(aerc *widgets.Aerc, args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: compose")
	}
	acct := aerc.SelectedAccount()
	composer := widgets.NewComposer(
		aerc.Config(), acct.AccountConfig(), acct.Worker(), nil)
	tab := aerc.NewTab(composer, "New email")
	composer.OnHeaderChange("Subject", func(subject string) {
		if subject == "" {
			tab.Name = "New email"
		} else {
			tab.Name = subject
		}
		tab.Content.Invalidate()
	})
	return nil
}
