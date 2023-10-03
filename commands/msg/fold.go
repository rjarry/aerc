package msg

import (
	"errors"
	"strings"

	"git.sr.ht/~rjarry/aerc/lib/ui"
)

type Fold struct{}

func init() {
	register(Fold{})
}

func (Fold) Aliases() []string {
	return []string{"fold", "unfold"}
}

func (Fold) Complete(args []string) []string {
	return nil
}

func (Fold) Execute(args []string) error {
	h := newHelper()
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
