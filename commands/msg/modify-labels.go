package msg

import (
	"errors"
	"time"

	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/widgets"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

type ModifyLabels struct{}

func init() {
	register(ModifyLabels{})
}

func (ModifyLabels) Aliases() []string {
	return []string{"modify-labels"}
}

func (ModifyLabels) Complete(aerc *widgets.Aerc, args []string) []string {
	return commands.GetLabels(aerc, args)
}

func (ModifyLabels) Execute(aerc *widgets.Aerc, args []string) error {
	changes := args[1:]
	if len(changes) == 0 {
		return errors.New("Usage: modify-labels <[+-]label> ...")
	}

	h := newHelper(aerc)
	store, err := h.store()
	if err != nil {
		return err
	}
	uids, err := h.markedOrSelectedUids()
	if err != nil {
		return err
	}

	var add, remove []string
	for _, l := range changes {
		switch l[0] {
		case '+':
			add = append(add, l[1:])
		case '-':
			remove = append(remove, l[1:])
		default:
			// if no operand is given assume add
			add = append(add, l)
		}
	}
	store.ModifyLabels(uids, add, remove, func(
		msg types.WorkerMessage,
	) {
		switch msg := msg.(type) {
		case *types.Done:
			aerc.PushStatus("labels updated", 10*time.Second)
			store.Marker().ClearVisualMark()
		case *types.Error:
			aerc.PushError(msg.Error.Error())
		}
	})
	return nil
}
