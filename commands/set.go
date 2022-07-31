package commands

import (
	"errors"
	"strings"

	"git.sr.ht/~rjarry/aerc/widgets"

	"github.com/go-ini/ini"
)

type Set struct{}

func setUsage() string {
	return "set <category>.<option> <value>"
}

func init() {
	register(Set{})
}

func (Set) Aliases() []string {
	return []string{"set"}
}

func (Set) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func SetCore(aerc *widgets.Aerc, args []string) error {
	if len(args) != 3 {
		return errors.New("Usage: " + setUsage())
	}

	config := aerc.Config()

	parameters := strings.Split(args[1], ".")

	if len(parameters) != 2 {
		return errors.New("Usage: " + setUsage())
	}

	category := parameters[0]
	option := parameters[1]
	value := args[2]

	new_file := ini.Empty()

	section, err := new_file.NewSection(category)
	if err != nil {
		return nil
	}

	if _, err := section.NewKey(option, value); err != nil {
		return err
	}

	if err := config.LoadConfig(new_file); err != nil {
		return err
	}

	// ensure any ui changes take effect
	aerc.Invalidate()

	return nil
}

func (Set) Execute(aerc *widgets.Aerc, args []string) error {
	return SetCore(aerc, args)
}
