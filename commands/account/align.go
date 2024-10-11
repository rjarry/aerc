package account

import (
	"errors"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/lib/ui"
)

type Align struct {
	Pos app.AlignPosition `opt:"pos" metavar:"top|center|bottom" action:"ParsePos" complete:"CompletePos" desc:"Position."`
}

func init() {
	commands.Register(Align{})
}

func (Align) Description() string {
	return "Align the message list view."
}

var posNames []string = []string{"top", "center", "bottom"}

func (a *Align) ParsePos(arg string) error {
	switch arg {
	case "top":
		a.Pos = app.AlignTop
	case "center":
		a.Pos = app.AlignCenter
	case "bottom":
		a.Pos = app.AlignBottom
	default:
		return errors.New("invalid alignment")
	}
	return nil
}

func (a *Align) CompletePos(arg string) []string {
	return commands.FilterList(posNames, arg, commands.QuoteSpace)
}

func (Align) Context() commands.CommandContext {
	return commands.MESSAGE_LIST
}

func (Align) Aliases() []string {
	return []string{"align"}
}

func (a Align) Execute(args []string) error {
	acct := app.SelectedAccount()
	if acct == nil {
		return errors.New("no account selected")
	}
	msgList := acct.Messages()
	if msgList == nil {
		return errors.New("no message list available")
	}
	msgList.AlignMessage(a.Pos)
	ui.Invalidate()

	return nil
}
