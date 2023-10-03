package compose

import (
	"git.sr.ht/~rjarry/aerc/app"
)

type AttachKey struct{}

func init() {
	register(AttachKey{})
}

func (AttachKey) Aliases() []string {
	return []string{"attach-key"}
}

func (AttachKey) Complete(args []string) []string {
	return nil
}

func (AttachKey) Execute(args []string) error {
	composer, _ := app.SelectedTabContent().(*app.Composer)
	return composer.SetAttachKey(!composer.AttachKey())
}
