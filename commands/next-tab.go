package commands

import (
	"git.sr.ht/~rjarry/aerc/app"
)

type NextPrevTab struct {
	Offset int `opt:"n" minus:"true" default:"1"`
}

func init() {
	Register(NextPrevTab{})
}

func (NextPrevTab) Description() string {
	return "Cycle to the previous or next tab."
}

func (NextPrevTab) Context() CommandContext {
	return GLOBAL
}

func (NextPrevTab) Aliases() []string {
	return []string{"next-tab", "prev-tab"}
}

func (np NextPrevTab) Execute(args []string) error {
	if np.Offset <= 0 {
		return nil
	}

	offset := np.Offset
	if args[0] == "prev-tab" {
		offset *= -1
	}

	app.SelectTabAtOffset(offset)
	app.UpdateStatus()
	return nil
}
