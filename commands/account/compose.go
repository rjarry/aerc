package account

import (
	"errors"

	"git.sr.ht/~sircmpwn/aerc/widgets"
)

func init() {
	register("compose", Compose)
}

// TODO: Accept arguments for default headers, message body
func Compose(aerc *widgets.Aerc, args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: compose")
	}
	acct := aerc.SelectedAccount()
	composer := widgets.NewComposer(
		aerc.Config(), acct.AccountConfig(), acct.Worker())
	tab := aerc.NewTab(composer, "New email")
	composer.OnSubjectChange(func(subject string) {
		if subject == "" {
			tab.Name = "New email"
		} else {
			tab.Name = subject
		}
		tab.Content.Invalidate()
	})
	return nil
}
