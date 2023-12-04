package msg

import (
	"fmt"

	"git.sr.ht/~rjarry/aerc/commands"
)

type Mark struct {
	All         bool `opt:"-a" aliases:"mark,unmark"`
	Toggle      bool `opt:"-t" aliases:"mark,unmark"`
	Visual      bool `opt:"-v" aliases:"mark,unmark"`
	VisualClear bool `opt:"-V" aliases:"mark,unmark"`
	Thread      bool `opt:"-T" aliases:"mark,unmark"`
}

func init() {
	commands.Register(Mark{})
}

func (Mark) Context() commands.CommandContext {
	return commands.MESSAGE
}

func (Mark) Aliases() []string {
	return []string{"mark", "unmark", "remark"}
}

func (m Mark) Execute(args []string) error {
	h := newHelper()
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

	if m.Thread && m.All {
		return fmt.Errorf("-a and -T are mutually exclusive")
	}

	if m.Thread && (m.Visual || m.VisualClear) {
		return fmt.Errorf("-v and -T are mutually exclusive")
	}
	if m.Visual && m.All {
		return fmt.Errorf("-a and -v are mutually exclusive")
	}

	switch args[0] {
	case "mark":
		var modFunc func(uint32)
		if m.Toggle {
			modFunc = marker.ToggleMark
		} else {
			modFunc = marker.Mark
		}
		switch {
		case m.All:
			uids := store.Uids()
			for _, uid := range uids {
				modFunc(uid)
			}
			return nil
		case m.Visual || m.VisualClear:
			marker.ToggleVisualMark(m.VisualClear)
			return nil
		default:
			if m.Thread {
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
		if m.Visual || m.VisualClear {
			return fmt.Errorf("visual mode not supported for this command")
		}

		switch {
		case m.All && m.Toggle:
			uids := store.Uids()
			for _, uid := range uids {
				marker.ToggleMark(uid)
			}
			return nil
		case m.All && !m.Toggle:
			marker.ClearVisualMark()
			return nil
		default:
			if m.Thread {
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
		marker.Remark()
		return nil
	}
	return nil // never reached
}
