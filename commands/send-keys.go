package commands

import (
	"strings"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/config"
	"github.com/gdamore/tcell/v2"
	"github.com/pkg/errors"
)

type SendKeys struct{}

func init() {
	register(SendKeys{})
}

func (SendKeys) Aliases() []string {
	return []string{"send-keys"}
}

func (SendKeys) Complete(args []string) []string {
	return nil
}

func (SendKeys) Execute(args []string) error {
	tab, ok := app.SelectedTabContent().(app.HasTerminal)
	if !ok {
		return errors.New("There is no terminal here")
	}

	term := tab.Terminal()
	if term == nil {
		return errors.New("The terminal is not active")
	}

	text2send := strings.Join(args[1:], "")
	keys2send, err := config.ParseKeyStrokes(text2send)
	if err != nil {
		return errors.Wrapf(err, "Unable to parse keystroke: '%s'", text2send)
	}

	for _, key := range keys2send {
		ev := tcell.NewEventKey(key.Key, key.Rune, key.Modifiers)
		term.Event(ev)
	}

	term.Invalidate()

	return nil
}
