package msg

import (
	"errors"
	"fmt"
	"strings"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/lib/ui"
)

type Fold struct{}

func init() {
	register(Fold{})
}

func (Fold) Aliases() []string {
	return []string{"fold", "unfold"}
}

func (Fold) Complete(aerc *app.Aerc, args []string) []string {
	return nil
}

func (Fold) Execute(aerc *app.Aerc, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("Usage: %s", args[0])
	}
	h := newHelper(aerc)
	store, err := h.store()
	if err != nil {
		return err
	}

	msg := store.Selected()
	if msg == nil {
		return errors.New("No message selected")
	}

	switch strings.ToLower(args[0]) {
	case "fold":
		err = store.Fold(msg.Uid)
	case "unfold":
		err = store.Unfold(msg.Uid)
	}
	ui.Invalidate()
	return err
}
