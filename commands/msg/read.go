package msg

import (
	"fmt"
	"sync"
	"time"

	"git.sr.ht/~sircmpwn/getopt"

	"git.sr.ht/~sircmpwn/aerc/lib"
	"git.sr.ht/~sircmpwn/aerc/models"
	"git.sr.ht/~sircmpwn/aerc/widgets"
	"git.sr.ht/~sircmpwn/aerc/worker/types"
)

type FlagMsg struct{}

func init() {
	register(FlagMsg{})
}

func (FlagMsg) Aliases() []string {
	return []string{"flag", "unflag", "read", "unread"}
}

func (FlagMsg) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

// If this was called as 'flag' or 'unflag', without the toggle (-t)
// option, then it will flag the corresponding messages with the given
// flag.  If the toggle option was given, it will individually toggle
// the given flag for the corresponding messages.
//
// If this was called as 'read' or 'unread', it has the same effect as
// 'flag' or 'unflag', respectively, but the 'Seen' flag is affected.
func (FlagMsg) Execute(aerc *widgets.Aerc, args []string) error {

	// The flag to change
	var flag models.Flag
	// User-readable name of the flag to change
	var flagName string
	// Whether to toggle the flag (true) or to enable/disable it (false)
	var toggle bool
	// Whether to enable (true) or disable (false) the flag
	enable := (args[0] == "read" || args[0] == "flag")
	// User-readable name for the action being performed
	var actionName string
	// Getopt option string, varies by command name
	var getoptString string
	// Help message to provide on parsing failure
	var helpMessage string
	// Used during parsing to prevent choosing a flag muliple times
	// A default flag will be used if this is false
	flagChosen := false

	if args[0] == "read" || args[0] == "unread" {
		flag = models.SeenFlag
		flagName = "read"
		getoptString = "t"
		helpMessage = "Usage: " + args[0] + " [-t]"
	} else { // 'flag' / 'unflag'
		flag = models.FlaggedFlag
		flagName = "flagged"
		getoptString = "tax:"
		helpMessage = "Usage: " + args[0] + " [-t] [-a | -x <flag>]"
	}

	opts, optind, err := getopt.Getopts(args, getoptString)
	if err != nil {
		return err
	}
	for _, opt := range opts {
		switch opt.Option {
		case 't':
			toggle = true
		case 'a':
			if flagChosen {
				return fmt.Errorf("Cannot choose a flag multiple times! " + helpMessage)
			}
			flag = models.AnsweredFlag
			flagName = "answered"
			flagChosen = true
		case 'x':
			if flagChosen {
				return fmt.Errorf("Cannot choose a flag multiple times! " + helpMessage)
			}
			// TODO: Support all flags?
			switch opt.Value {
			case "Seen":
				flag = models.SeenFlag
				flagName = "seen"
			case "Answered":
				flag = models.AnsweredFlag
				flagName = "answered"
			case "Flagged":
				flag = models.FlaggedFlag
				flagName = "flagged"
			default:
				return fmt.Errorf("Unknown / Prohibited flag \"%v\"", opt.Value)
			}
			flagChosen = true
		}
	}
	if toggle {
		actionName = "Toggling"
	} else if enable {
		actionName = "Setting"
	} else {
		actionName = "Unsetting"
	}
	if optind != len(args) {
		// Any non-option arguments: Error
		return fmt.Errorf(helpMessage)
	}

	h := newHelper(aerc)
	store, err := h.store()
	if err != nil {
		return err
	}

	// UIDs of messages to enable or disable the flag for.
	var toEnable []uint32
	var toDisable []uint32

	if toggle {
		// If toggling, split messages into those that need to
		// be enabled / disabled.
		msgs, err := h.messages()
		if err != nil {
			return err
		}
		for _, m := range msgs {
			var enabled bool
			for _, mFlag := range m.Flags {
				if mFlag == flag {
					enabled = true
					break
				}
			}
			if enabled {
				toDisable = append(toDisable, m.Uid)
			} else {
				toEnable = append(toEnable, m.Uid)
			}
		}
	} else {
		msgUids, err := h.markedOrSelectedUids()
		if err != nil {
			return err
		}
		if enable {
			toEnable = msgUids
		} else {
			toDisable = msgUids
		}
	}

	var wg sync.WaitGroup
	success := true

	if len(toEnable) != 0 {
		submitFlagChange(aerc, store, toEnable, flag, true, &wg, &success)
	}
	if len(toDisable) != 0 {
		submitFlagChange(aerc, store, toDisable, flag, false, &wg, &success)
	}

	// We need to do flagging in the background, else we block the main thread
	go func() {
		wg.Wait()
		if success {
			aerc.PushStatus(actionName+" flag '"+flagName+"' successful", 10*time.Second)
		}
	}()

	return nil
}

func submitFlagChange(aerc *widgets.Aerc, store *lib.MessageStore,
	uids []uint32, flag models.Flag, newState bool,
	wg *sync.WaitGroup, success *bool) {
	store.Flag(uids, flag, newState, func(msg types.WorkerMessage) {
		wg.Add(1)
		switch msg := msg.(type) {
		case *types.Done:
			wg.Done()
		case *types.Error:
			aerc.PushError(msg.Error.Error())
			*success = false
			wg.Done()
		}
	})
}
