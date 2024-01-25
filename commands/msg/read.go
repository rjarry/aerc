package msg

import (
	"fmt"
	"strings"
	"time"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

type FlagMsg struct {
	Toggle   bool         `opt:"-t"`
	Answered bool         `opt:"-a" aliases:"flag,unflag"`
	Flag     models.Flags `opt:"-x" aliases:"flag,unflag" action:"ParseFlag" complete:"CompleteFlag"`
	FlagName string
}

func init() {
	commands.Register(FlagMsg{})
}

func (FlagMsg) Context() commands.CommandContext {
	return commands.MESSAGE
}

func (FlagMsg) Aliases() []string {
	return []string{"flag", "unflag", "read", "unread"}
}

func (f *FlagMsg) ParseFlag(arg string) error {
	switch strings.ToLower(arg) {
	case "seen":
		f.Flag = models.SeenFlag
		f.FlagName = "seen"
	case "answered":
		f.Flag = models.AnsweredFlag
		f.FlagName = "answered"
	case "flagged":
		f.Flag = models.FlaggedFlag
		f.FlagName = "flagged"
	case "draft":
		f.Flag = models.DraftFlag
		f.FlagName = "draft"
	default:
		return fmt.Errorf("Unknown flag %q", arg)
	}
	return nil
}

var validFlags = []string{"seen", "answered", "flagged", "draft"}

func (*FlagMsg) CompleteFlag(arg string) []string {
	return commands.FilterList(validFlags, arg, nil)
}

// If this was called as 'flag' or 'unflag', without the toggle (-t)
// option, then it will flag the corresponding messages with the given
// flag.  If the toggle option was given, it will individually toggle
// the given flag for the corresponding messages.
//
// If this was called as 'read' or 'unread', it has the same effect as
// 'flag' or 'unflag', respectively, but the 'Seen' flag is affected.
func (f FlagMsg) Execute(args []string) error {
	// User-readable name for the action being performed
	var actionName string

	switch args[0] {
	case "read", "unread":
		f.Flag = models.SeenFlag
		f.FlagName = "seen"
	case "flag", "unflag":
		if f.Answered {
			f.Flag = models.AnsweredFlag
			f.FlagName = "answered"
		}
		if f.Flag == 0 {
			f.Flag = models.FlaggedFlag
			f.FlagName = "flagged"
		}
	}

	h := newHelper()
	store, err := h.store()
	if err != nil {
		return err
	}

	// UIDs of messages to enable or disable the flag for.
	var toEnable []uint32
	var toDisable []uint32

	if f.Toggle {
		// If toggling, split messages into those that need to
		// be enabled / disabled.
		msgs, err := h.messages()
		if err != nil {
			return err
		}
		for _, m := range msgs {
			if m.Flags.Has(f.Flag) {
				toDisable = append(toDisable, m.Uid)
			} else {
				toEnable = append(toEnable, m.Uid)
			}
		}
		actionName = "Toggling"
	} else {
		msgUids, err := h.markedOrSelectedUids()
		if err != nil {
			return err
		}
		switch args[0] {
		case "read", "flag":
			toEnable = msgUids
			actionName = "Setting"
		default:
			toDisable = msgUids
			actionName = "Unsetting"
		}
	}

	status := fmt.Sprintf("%s flag %q successful", actionName, f.FlagName)

	if len(toEnable) != 0 {
		store.Flag(toEnable, f.Flag, true, func(msg types.WorkerMessage) {
			switch msg := msg.(type) {
			case *types.Done:
				app.PushStatus(status, 10*time.Second)
				store.Marker().ClearVisualMark()
			case *types.Error:
				app.PushError(msg.Error.Error())
			}
		})
	}
	if len(toDisable) != 0 {
		store.Flag(toDisable, f.Flag, false, func(msg types.WorkerMessage) {
			switch msg := msg.(type) {
			case *types.Done:
				app.PushStatus(status, 10*time.Second)
				store.Marker().ClearVisualMark()
			case *types.Error:
				app.PushError(msg.Error.Error())
			}
		})
	}
	return nil
}
