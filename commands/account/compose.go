package account

import (
	"errors"

	"github.com/mattn/go-runewidth"

	"git.sr.ht/~sircmpwn/aerc2/widgets"
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
	composer := widgets.NewComposer(aerc.Config(), acct.AccountConfig())
	// TODO: Change tab name when message subject changes
	aerc.NewTab(composer, runewidth.Truncate(
		"New email", 32, "…"))
	return nil
}


