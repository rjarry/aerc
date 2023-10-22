package compose

import (
	"git.sr.ht/~rjarry/aerc/app"
)

type NextPrevField struct{}

func init() {
	register(NextPrevField{})
}

func (NextPrevField) Aliases() []string {
	return []string{"next-field", "prev-field"}
}

func (NextPrevField) Execute(args []string) error {
	composer, _ := app.SelectedTabContent().(*app.Composer)
	if args[0] == "prev-field" {
		composer.PrevField()
	} else {
		composer.NextField()
	}
	return nil
}
