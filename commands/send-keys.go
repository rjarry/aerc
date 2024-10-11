package commands

import (
	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rockorager/vaxis"
	"github.com/pkg/errors"
)

type SendKeys struct {
	Keys string `opt:"..."`
}

func init() {
	Register(SendKeys{})
}

func (SendKeys) Description() string {
	return "Send keystrokes to the currently visible terminal."
}

func (SendKeys) Context() CommandContext {
	return GLOBAL
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
		ev := vaxis.Key{
			Keycode:   key.Key,
			Modifiers: key.Modifiers,
		}
		term.Event(ev)
	}

	term.Invalidate()

	return nil
}
