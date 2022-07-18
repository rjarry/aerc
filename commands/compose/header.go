package compose

import (
	"errors"
	"fmt"
	"strings"

	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/widgets"
	"git.sr.ht/~sircmpwn/getopt"
)

type Header struct{}

var (
	headers = []string{
		"From",
		"To",
		"Cc",
		"Bcc",
		"Subject",
		"Comments",
		"Keywords",
	}
)

func init() {
	register(Header{})
}

func (Header) Aliases() []string {
	return []string{"header"}
}

func (Header) Complete(aerc *widgets.Aerc, args []string) []string {
	return commands.CompletionFromList(aerc, headers, args)
}

func (Header) Execute(aerc *widgets.Aerc, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("Usage: %s [-f] field [value]", args[0])
	}

	opts, optind, err := getopt.Getopts(args, "f")
	if err != nil {
		return err
	}

	if len(args) < optind+1 {
		return errors.New("command parsing failed")
	}

	var (
		force bool = false
	)
	for _, opt := range opts {
		switch opt.Option {
		case 'f':
			force = true
		}
	}

	composer, _ := aerc.SelectedTabContent().(*widgets.Composer)

	args[optind] = strings.TrimRight(args[optind], ":")

	value := strings.Join(args[optind+1:], " ")

	if !force {
		headers, err := composer.PrepareHeader()
		if err != nil {
			return err
		}

		if headers.Has(args[optind]) && value != "" {
			return fmt.Errorf("Header %s already exists", args[optind])
		}
	}

	composer.AddEditor(args[optind], value, false)

	return nil
}
