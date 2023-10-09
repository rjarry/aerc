package compose

import (
	"errors"
	"time"

	"git.sr.ht/~rjarry/aerc/app"
)

type Sign struct{}

func init() {
	register(Sign{})
}

func (Sign) Aliases() []string {
	return []string{"sign"}
}

func (Sign) Complete(args []string) []string {
	return nil
}

func (Sign) Execute(args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: sign")
	}

	composer, _ := app.SelectedTabContent().(*app.Composer)

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

	app.PushStatus(statusline, 10*time.Second)

	return nil
}
