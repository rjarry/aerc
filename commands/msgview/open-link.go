package msgview

import (
	"fmt"
	"net/url"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/log"
)

type OpenLink struct {
	Url *url.URL `opt:"url" action:"ParseUrl" complete:"CompleteUrl"`
	Cmd string   `opt:"..." required:"false"`
}

func init() {
	register(OpenLink{})
}

func (OpenLink) Aliases() []string {
	return []string{"open-link"}
}

func (*OpenLink) CompleteUrl(arg string) []string {
	mv := app.SelectedTabContent().(*app.MessageViewer)
	if mv != nil {
		if p := mv.SelectedMessagePart(); p != nil {
			return commands.CompletionFromList(p.Links, arg)
		}
	}
	return nil
}

func (o *OpenLink) ParseUrl(arg string) error {
	u, err := url.Parse(arg)
	if err != nil {
		return err
	}
	o.Url = u
	return nil
}

func (o OpenLink) Execute(args []string) error {
	mime := fmt.Sprintf("x-scheme-handler/%s", o.Url.Scheme)
	go func() {
		defer log.PanicHandler()
		if err := lib.XDGOpenMime(o.Url.String(), mime, o.Cmd); err != nil {
			app.PushError("open-link: " + err.Error())
		}
	}()
	return nil
}
