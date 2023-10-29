package msg

import (
	"errors"
	"strings"

	"git.sr.ht/~rjarry/aerc/lib/ui"
)

type Fold struct {
	Toggle bool `opt:"-t" aliases:"fold,unfold"`
}

func init() {
	register(Fold{})
}

func (Fold) Aliases() []string {
	return []string{"fold", "unfold"}
}

func (f Fold) Execute(args []string) error {
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
		err = store.Fold(msg.Uid, f.Toggle)
	case "unfold":
		err = store.Unfold(msg.Uid, f.Toggle)
	}
	ui.Invalidate()
	return err
}
