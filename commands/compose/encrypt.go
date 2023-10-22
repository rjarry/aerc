package compose

import (
	"git.sr.ht/~rjarry/aerc/app"
)

type Encrypt struct{}

func init() {
	register(Encrypt{})
}

func (Encrypt) Aliases() []string {
	return []string{"encrypt"}
}

func (Encrypt) Execute(args []string) error {
	composer, _ := app.SelectedTabContent().(*app.Composer)
	composer.SetEncrypt(!composer.Encrypt())
	return nil
}
