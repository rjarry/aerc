package compose

import (
	"errors"

	"git.sr.ht/~rjarry/aerc/app"
)

type Encrypt struct{}

func init() {
	register(Encrypt{})
}

func (Encrypt) Aliases() []string {
	return []string{"encrypt"}
}

func (Encrypt) Complete(aerc *app.Aerc, args []string) []string {
	return nil
}

func (Encrypt) Execute(aerc *app.Aerc, args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: encrypt")
	}

	composer, _ := aerc.SelectedTabContent().(*app.Composer)

	composer.SetEncrypt(!composer.Encrypt())
	return nil
}
