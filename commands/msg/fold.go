package msg

import (
	"errors"

	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/lib/ui"
)

type Fold struct {
	All    bool `opt:"-a" desc:"Fold/unfold all threads."`
	Toggle bool `opt:"-t" desc:"Toggle between folded/unfolded."`
}

func init() {
	commands.Register(Fold{})
}

func (Fold) Description() string {
	return "Collapse or expand the thread children of the selected message."
}

func (Fold) Context() commands.CommandContext {
	return commands.MESSAGE_LIST | commands.MESSAGE_VIEWER
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

	if f.All {
		point := store.SelectedUid()
		uids := store.Uids()
		for _, uid := range uids {
			t, err := store.Thread(uid)
			if err == nil && t.Parent == nil {
				switch args[0] {
				case "fold":
					err = store.Fold(uid, f.Toggle)
				case "unfold":
					err = store.Unfold(uid, f.Toggle)
				}
			}
			if err != nil {
				return err
			}
		}
		store.Select(point)
		ui.Invalidate()
		return err
	}

	msg := store.Selected()
	if msg == nil {
		return errors.New("No message selected")
	}

	switch args[0] {
	case "fold":
		err = store.Fold(msg.Uid, f.Toggle)
	case "unfold":
		err = store.Unfold(msg.Uid, f.Toggle)
	}
	ui.Invalidate()
	return err
}
