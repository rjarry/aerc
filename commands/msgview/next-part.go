package msgview

import (
	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
)

type NextPrevPart struct {
	Offset int `opt:"n" default:"1"`
}

func init() {
	commands.Register(NextPrevPart{})
}

func (NextPrevPart) Description() string {
	return "Cycle between message parts being shown."
}

func (NextPrevPart) Context() commands.CommandContext {
	return commands.MESSAGE_VIEWER
}

func (NextPrevPart) Aliases() []string {
	return []string{"next-part", "prev-part"}
}

func (np NextPrevPart) Execute(args []string) error {
	mv, _ := app.SelectedTabContent().(*app.MessageViewer)
	for n := 0; n < np.Offset; n++ {
		if args[0] == "prev-part" {
			mv.PreviousPart()
		} else {
			mv.NextPart()
		}
	}
	return nil
}
