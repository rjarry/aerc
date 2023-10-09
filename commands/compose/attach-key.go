package compose

import (
	"errors"

	"git.sr.ht/~rjarry/aerc/app"
)

type AttachKey struct{}

func init() {
	register(AttachKey{})
}

func (AttachKey) Aliases() []string {
	return []string{"attach-key"}
}

func (AttachKey) Complete(aerc *app.Aerc, args []string) []string {
	return nil
}

func (AttachKey) Execute(aerc *app.Aerc, args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: attach-key")
	}

	composer, _ := aerc.SelectedTabContent().(*app.Composer)

	return composer.SetAttachKey(!composer.AttachKey())
}
