package msg

import (
	"fmt"

	"git.sr.ht/~rjarry/aerc/widgets"
	"git.sr.ht/~sircmpwn/getopt"
)

type Mark struct{}

func init() {
	register(Mark{})
}

func (Mark) Aliases() []string {
	return []string{"mark", "unmark"}
}

func (Mark) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (Mark) Execute(aerc *widgets.Aerc, args []string) error {
	h := newHelper(aerc)
	selected, err := h.msgProvider.SelectedMessage()
	if err != nil {
		return err
	}
	store, err := h.store()
	if err != nil {
		return err
	}
	opts, _, err := getopt.Getopts(args, "atv")
	if err != nil {
		return err
	}
	var all bool
	var toggle bool
	var visual bool
	for _, opt := range opts {
		switch opt.Option {
		case 'a':
			all = true
		case 'v':
			visual = true
		case 't':
			toggle = true
		}
	}

	switch args[0] {
	case "mark":
		if all && visual {
			return fmt.Errorf("-a and -v are mutually exclusive")
		}

		var modFunc func(uint32)
		if toggle {
			modFunc = store.ToggleMark
		} else {
			modFunc = store.Mark
		}
		if all {
			uids := store.Uids()
			for _, uid := range uids {
				modFunc(uid)
			}
			return nil
		} else if visual {
			store.ToggleVisualMark()
			return nil
		} else {
			modFunc(selected.Uid)
			return nil
		}

	case "unmark":
		if visual {
			return fmt.Errorf("visual mode not supported for this command")
		}

		if all && toggle {
			uids := store.Uids()
			for _, uid := range uids {
				store.ToggleMark(uid)
			}
			return nil
		} else if all && !toggle {
			store.ClearVisualMark()
			return nil
		} else {
			store.Unmark(selected.Uid)
			return nil
		}
	}
	return nil // never reached
}
