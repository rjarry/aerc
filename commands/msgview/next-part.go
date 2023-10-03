package msgview

import (
	"git.sr.ht/~rjarry/aerc/app"
)

type NextPrevPart struct {
	Offset int `opt:"n" default:"1"`
}

func init() {
	register(NextPrevPart{})
}

func (NextPrevPart) Aliases() []string {
	return []string{"next-part", "prev-part"}
}

func (NextPrevPart) Complete(args []string) []string {
	return nil
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
