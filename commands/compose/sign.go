package compose

import (
	"errors"
	"time"

	"git.sr.ht/~rjarry/aerc/widgets"
)

type Sign struct{}

func init() {
	register(Sign{})
}

func (Sign) Aliases() []string {
	return []string{"sign"}
}

func (Sign) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (Sign) Execute(aerc *widgets.Aerc, args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: sign")
	}

	composer, _ := aerc.SelectedTabContent().(*widgets.Composer)

	err := composer.SetSign(!composer.Sign())
	if err != nil {
		return err
	}

	var statusline string

	if composer.Sign() {
		statusline = "Message will be signed."
	} else {
		statusline = "Message will not be signed."
	}

	aerc.PushStatus(statusline, 10*time.Second)

	return nil
}
