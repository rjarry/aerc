package compose

import (
	"fmt"
	"strings"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
)

type Header struct {
	Force  bool   `opt:"-f" desc:"Overwrite any existing header."`
	Remove bool   `opt:"-d" desc:"Remove the header instead of adding it."`
	Name   string `opt:"name" complete:"CompleteHeaders" desc:"Header name."`
	Value  string `opt:"..." required:"false"`
}

var headers = []string{
	"From",
	"To",
	"Cc",
	"Bcc",
	"Subject",
	"Comments",
	"Keywords",
}

func init() {
	commands.Register(Header{})
}

func (Header) Description() string {
	return "Add or remove the specified email header."
}

func (Header) Context() commands.CommandContext {
	return commands.COMPOSE_EDIT | commands.COMPOSE_REVIEW
}

func (Header) Aliases() []string {
	return []string{"header"}
}

func (Header) Options() string {
	return "fd"
}

func (*Header) CompleteHeaders(arg string) []string {
	return commands.FilterList(headers, arg, commands.QuoteSpace)
}

func (h Header) Execute(args []string) error {
	composer, _ := app.SelectedTabContent().(*app.Composer)

	name := strings.TrimRight(h.Name, ":")

	if h.Remove {
		return composer.DelEditor(name)
	}

	if !h.Force {
		headers, err := composer.PrepareHeader()
		if err != nil {
			return err
		}
		if headers.Get(name) != "" && h.Value != "" {
			return fmt.Errorf(
				"Header %s is already set to %q (use -f to overwrite)",
				name, headers.Get(name))
		}
	}

	return composer.AddEditor(name, h.Value, false)
}
