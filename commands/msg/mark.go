package msg

import (
	"fmt"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~sircmpwn/getopt"
)

type Mark struct{}

func init() {
	register(Mark{})
}

func (Mark) Aliases() []string {
	return []string{"mark", "unmark", "remark"}
}

func (Mark) Complete(aerc *app.Aerc, args []string) []string {
	return nil
}

func (Mark) Execute(aerc *app.Aerc, args []string) error {
	h := newHelper(aerc)
	OnSelectedMessage := func(fn func(uint32)) error {
		if fn == nil {
			return fmt.Errorf("no operation selected")
		}
		selected, err := h.msgProvider.SelectedMessage()
		if err != nil {
			return err
		}
		fn(selected.Uid)
		return nil
	}
	store, err := h.store()
	if err != nil {
		return err
	}
	marker := store.Marker()
	opts, _, err := getopt.Getopts(args, "atvVT")
	if err != nil {
		return err
	}
	var all bool
	var toggle bool
	var visual bool
	var clearVisual bool
	var thread bool
	for _, opt := range opts {
		switch opt.Option {
		case 'a':
			all = true
		case 'v':
			visual = true
			clearVisual = true
		case 'V':
			visual = true
		case 't':
			toggle = true
		case 'T':
			thread = true
		}
	}

	if thread && all {
		return fmt.Errorf("-a and -T are mutually exclusive")
	}

	if thread && visual {
		return fmt.Errorf("-v and -T are mutually exclusive")
	}

	switch args[0] {
	case "mark":
		if all && visual {
			return fmt.Errorf("-a and -v are mutually exclusive")
		}

		var modFunc func(uint32)
		if toggle {
			modFunc = marker.ToggleMark
		} else {
			modFunc = marker.Mark
		}
		switch {
		case all:
			uids := store.Uids()
			for _, uid := range uids {
				modFunc(uid)
			}
			return nil
		case visual:
			marker.ToggleVisualMark(clearVisual)
			return nil
		default:
			if thread {
				threadPtr, err := store.SelectedThread()
				if err != nil {
					return err
				}
				for _, uid := range threadPtr.Root().Uids() {
					modFunc(uid)
				}
			} else {
				return OnSelectedMessage(modFunc)
			}
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
				marker.ToggleMark(uid)
			}
			return nil
		case all && !toggle:
			marker.ClearVisualMark()
			return nil
		default:
			if thread {
				threadPtr, err := store.SelectedThread()
				if err != nil {
					return err
				}
				for _, uid := range threadPtr.Root().Uids() {
					marker.Unmark(uid)
				}
			} else {
				return OnSelectedMessage(marker.Unmark)
			}
			return nil
		}
	case "remark":
		if all || visual || toggle || thread {
			return fmt.Errorf("Usage: :remark")
		}
		marker.Remark()
		return nil
	}
	return nil // never reached
}
