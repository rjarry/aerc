package compose

import (
	"time"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
)

type Sign struct{}

func init() {
	commands.Register(Sign{})
}

func (Sign) Context() commands.CommandContext {
	return commands.COMPOSE
}

func (Sign) Aliases() []string {
	return []string{"sign"}
}

func (Sign) Execute(args []string) error {
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
