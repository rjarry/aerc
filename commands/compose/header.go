package compose

import (
	"fmt"
	"strings"

	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/widgets"
	"git.sr.ht/~sircmpwn/getopt"
)

type Header struct{}

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
	register(Header{})
}

func (Header) Aliases() []string {
	return []string{"header"}
}

func (Header) Options() string {
	return "fd"
}

func (Header) Complete(aerc *widgets.Aerc, args []string) []string {
	return commands.CompletionFromList(aerc, headers, args)
}

func (h Header) Execute(aerc *widgets.Aerc, args []string) error {
	opts, optind, err := getopt.Getopts(args, h.Options())
	args = args[optind:]
	if err == nil && len(args) < 1 {
		err = fmt.Errorf("not enough arguments")
	}
	if err != nil {
		return fmt.Errorf("%w. usage: header [-fd] <name> [<value>]", err)
	}

	var force bool = false
	var remove bool = false
	for _, opt := range opts {
		switch opt.Option {
		case 'f':
			force = true
		case 'd':
			remove = true
		}
	}

	composer, _ := aerc.SelectedTabContent().(*widgets.Composer)

	name := strings.TrimRight(args[0], ":")

	if remove {
		return composer.DelEditor(name)
	}

	value := strings.Join(args[1:], " ")

	if !force {
		headers, err := composer.PrepareHeader()
		if err != nil {
			return err
		}
		if headers.Get(name) != "" && value != "" {
			return fmt.Errorf(
				"Header %s is already set to %q (use -f to overwrite)",
				name, headers.Get(name))
		}
	}

	return composer.AddEditor(name, value, false)
}
