package compose

import (
	"fmt"

	"git.sr.ht/~rjarry/aerc/app"
)

type NextPrevField struct{}

func init() {
	register(NextPrevField{})
}

func (NextPrevField) Aliases() []string {
	return []string{"next-field", "prev-field"}
}

func (NextPrevField) Complete(aerc *app.Aerc, args []string) []string {
	return nil
}

func (NextPrevField) Execute(aerc *app.Aerc, args []string) error {
	if len(args) > 2 {
		return nextPrevFieldUsage(args[0])
	}
	composer, _ := aerc.SelectedTabContent().(*app.Composer)
	if args[0] == "prev-field" {
		composer.PrevField()
	} else {
		composer.NextField()
	}
	return nil
}

func nextPrevFieldUsage(cmd string) error {
	return fmt.Errorf("Usage: %s", cmd)
}
