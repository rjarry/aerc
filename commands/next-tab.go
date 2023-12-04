package commands

import (
	"git.sr.ht/~rjarry/aerc/app"
)

type NextPrevTab struct {
	Offset int `opt:"n" default:"1"`
}

func init() {
	Register(NextPrevTab{})
}

func (NextPrevTab) Context() CommandContext {
	return GLOBAL
}

func (NextPrevTab) Aliases() []string {
	return []string{"next-tab", "prev-tab"}
}

func (np NextPrevTab) Execute(args []string) error {
	for n := 0; n < np.Offset; n++ {
		if args[0] == "prev-tab" {
			app.PrevTab()
		} else {
			app.NextTab()
		}
	}
	app.UpdateStatus()
	return nil
}
