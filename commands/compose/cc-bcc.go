package compose

import (
	"fmt"
	"strings"

	"git.sr.ht/~sircmpwn/aerc/widgets"
)

type CC struct{}

func init() {
	register(CC{})
}

func (_ CC) Aliases() []string {
	return []string{"cc", "bcc"}
}

func (_ CC) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (_ CC) Execute(aerc *widgets.Aerc, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("Usage: %s <addresses>", args[0])
	}
	addrs := strings.Join(args[1:], " ")
	composer, _ := aerc.SelectedTab().(*widgets.Composer)

	switch args[0] {
	case "cc":
		composer.AddEditor("Cc", addrs)
	case "bcc":
		composer.AddEditor("Bcc", addrs)
	}

	return nil
}
