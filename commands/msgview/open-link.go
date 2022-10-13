package msgview

import (
	"errors"

	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/widgets"
)

type OpenLink struct{}

func init() {
	register(OpenLink{})
}

func (OpenLink) Aliases() []string {
	return []string{"open-link"}
}

func (OpenLink) Complete(aerc *widgets.Aerc, args []string) []string {
	mv := aerc.SelectedTabContent().(*widgets.MessageViewer)
	if mv != nil {
		if p := mv.SelectedMessagePart(); p != nil {
			return commands.CompletionFromList(aerc, p.Links, args)
		}
	}
	return nil
}

func (OpenLink) Execute(aerc *widgets.Aerc, args []string) error {
	if len(args) != 2 {
		return errors.New("Usage: open-link <url>")
	}
	go func() {
		if err := lib.XDGOpen(args[1]); err != nil {
			aerc.PushError("open-link: " + err.Error())
		}
	}()
	return nil
}
