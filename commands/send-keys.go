package commands

import (
	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/config"
	"github.com/gdamore/tcell/v2"
	"github.com/pkg/errors"
)

type SendKeys struct {
	Keys string `opt:"..."`
}

func init() {
	register(SendKeys{})
}

func (SendKeys) Aliases() []string {
	return []string{"send-keys"}
}

func (s SendKeys) Execute(args []string) error {
	tab, ok := app.SelectedTabContent().(app.HasTerminal)
	if !ok {
		return errors.New("There is no terminal here")
	}

	term := tab.Terminal()
	if term == nil {
		return errors.New("The terminal is not active")
	}

	keys2send, err := config.ParseKeyStrokes(s.Keys)
	if err != nil {
		return errors.Wrapf(err, "Unable to parse keystroke: %q", s.Keys)
	}

	for _, key := range keys2send {
		ev := tcell.NewEventKey(key.Key, key.Rune, key.Modifiers)
		term.Event(ev)
	}

	term.Invalidate()

	return nil
}
