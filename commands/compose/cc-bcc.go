package compose

import (
	"strings"

	"git.sr.ht/~rjarry/aerc/app"
)

type CC struct{}

func init() {
	register(CC{})
}

func (CC) Aliases() []string {
	return []string{"cc", "bcc"}
}

func (CC) Complete(aerc *app.Aerc, args []string) []string {
	return nil
}

func (CC) Execute(aerc *app.Aerc, args []string) error {
	var addrs string
	if len(args) > 1 {
		addrs = strings.Join(args[1:], " ")
	}
	composer, _ := aerc.SelectedTabContent().(*app.Composer)

	switch args[0] {
	case "cc":
		return composer.AddEditor("Cc", addrs, true)
	case "bcc":
		return composer.AddEditor("Bcc", addrs, true)
	}

	return nil
}
