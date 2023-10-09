package msgview

import (
	"errors"
	"fmt"
	"net/url"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/log"
)

type OpenLink struct{}

func init() {
	register(OpenLink{})
}

func (OpenLink) Aliases() []string {
	return []string{"open-link"}
}

func (OpenLink) Complete(args []string) []string {
	mv := app.SelectedTabContent().(*app.MessageViewer)
	if mv != nil {
		if p := mv.SelectedMessagePart(); p != nil {
			return commands.CompletionFromList(p.Links, args)
		}
	}
	return nil
}

func (OpenLink) Execute(args []string) error {
	if len(args) < 2 {
		return errors.New("Usage: open-link <url> [program [args...]]")
	}
	u, err := url.Parse(args[1])
	if err != nil {
		return err
	}
	mime := fmt.Sprintf("x-scheme-handler/%s", u.Scheme)
	go func() {
		defer log.PanicHandler()
		if err := lib.XDGOpenMime(args[1], mime, args[2:]); err != nil {
			app.PushError("open-link: " + err.Error())
		}
	}()
	return nil
}
