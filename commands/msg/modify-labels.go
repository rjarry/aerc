package msg

import (
	"time"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

type ModifyLabels struct {
	Labels []string `opt:"..." metavar:"[+-]<label>" complete:"CompleteLabels"`
}

func init() {
	register(ModifyLabels{})
}

func (ModifyLabels) Aliases() []string {
	return []string{"modify-labels", "tag"}
}

func (*ModifyLabels) CompleteLabels(arg string) []string {
	return commands.GetLabels(arg)
}

func (m ModifyLabels) Execute(args []string) error {
	h := newHelper()
	store, err := h.store()
	if err != nil {
		return err
	}
	uids, err := h.markedOrSelectedUids()
	if err != nil {
		return err
	}

	var add, remove []string
	for _, l := range m.Labels {
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
			app.PushStatus("labels updated", 10*time.Second)
			store.Marker().ClearVisualMark()
		case *types.Error:
			app.PushError(msg.Error.Error())
		}
	})
	return nil
}
