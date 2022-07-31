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
	return []string{"mark", "unmark", "remark"}
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
		switch {
		case all:
			uids := store.Uids()
			for _, uid := range uids {
				modFunc(uid)
			}
			return nil
		case visual:
			store.ToggleVisualMark()
			return nil
		default:
			modFunc(selected.Uid)
			return nil
		}

	case "unmark":
		if visual {
			return fmt.Errorf("visual mode not supported for this command")
		}

		switch {
		case all && toggle:
			uids := store.Uids()
			for _, uid := range uids {
				store.ToggleMark(uid)
			}
			return nil
		case all && !toggle:
			store.ClearVisualMark()
			return nil
		default:
			store.Unmark(selected.Uid)
			return nil
		}
	case "remark":
		if all || visual || toggle {
			return fmt.Errorf("Usage: :remark")
		}
		store.Remark()
		return nil
	}
	return nil // never reached
}
