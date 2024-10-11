package compose

import (
	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
)

type CC struct {
	Recipients string `opt:"recipients" complete:"CompleteAddress"`
}

func init() {
	commands.Register(CC{})
}

func (CC) Description() string {
	return "Add the given address(es) to the Cc or Bcc header."
}

func (CC) Context() commands.CommandContext {
	return commands.COMPOSE
}

func (CC) Aliases() []string {
	return []string{"cc", "bcc"}
}

func (*CC) CompleteAddress(arg string) []string {
	return commands.GetAddress(arg)
}

func (c CC) Execute(args []string) error {
	composer, _ := app.SelectedTabContent().(*app.Composer)

	switch args[0] {
	case "cc":
		return composer.AddEditor("Cc", c.Recipients, true)
	case "bcc":
		return composer.AddEditor("Bcc", c.Recipients, true)
	}

	return nil
}
