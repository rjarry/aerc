package compose

import (
	"errors"

	"git.sr.ht/~rjarry/aerc/widgets"
)

type Encrypt struct{}

func init() {
	register(Encrypt{})
}

func (Encrypt) Aliases() []string {
	return []string{"encrypt"}
}

func (Encrypt) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (Encrypt) Execute(aerc *widgets.Aerc, args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: encrypt")
	}

	composer, _ := aerc.SelectedTab().(*widgets.Composer)

	composer.SetEncrypt(!composer.Encrypt())
	return nil
}
