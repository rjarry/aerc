package compose

import (
	"git.sr.ht/~rjarry/aerc/app"
)

type CC struct {
	Recipients string `opt:"recipients"`
}

func init() {
	register(CC{})
}

func (CC) Aliases() []string {
	return []string{"cc", "bcc"}
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
