package msg

import (
	"slices"
	"strings"
	"time"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

type ModifyLabels struct {
	Labels []string `opt:"..." minus:"true" metavar:"[+-!]<label>" complete:"CompleteLabels" desc:"Message label."`
}

func init() {
	commands.Register(ModifyLabels{})
}

func (ModifyLabels) Description() string {
	return "Modify message labels."
}

func (ModifyLabels) Context() commands.CommandContext {
	return commands.MESSAGE_LIST | commands.MESSAGE_VIEWER
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

	var add, remove, toggle []string
	for _, l := range m.Labels {
		switch l[0] {
		case '+':
			add = append(add, l[1:])
		case '-':
			remove = append(remove, l[1:])
		case '!':
			toggle = append(toggle, l[1:])
		default:
			// if no operand is given assume add
			add = append(add, l)
		}
	}

	// Compute resolved add/remove for status message only
	resolvedAdd := make([]string, len(add))
	copy(resolvedAdd, add)
	resolvedRemove := make([]string, len(remove))
	copy(resolvedRemove, remove)
	currentLabels := store.Selected().Labels
	for _, tag := range toggle {
		if slices.Contains(add, tag) || slices.Contains(remove, tag) {
			continue
		}
		if slices.Contains(currentLabels, tag) {
			resolvedRemove = append(resolvedRemove, tag)
		} else {
			resolvedAdd = append(resolvedAdd, tag)
		}
	}

	store.ModifyLabels(uids, add, remove, toggle, func(
		msg types.WorkerMessage,
	) {
		switch msg := msg.(type) {
		case *types.Done:
			synonym := "Labels"
			if args[0] == "tag" {
				synonym = "Tags"
			}
			app.PushStatus(synonym+" updated: "+tagChanges(resolvedAdd, resolvedRemove), 10*time.Second)
			store.Marker().ClearVisualMark()
		case *types.Error:
			app.PushError(msg.Error.Error())
		}
	})
	return nil
}

func tagChanges(add, remove []string) string {
	var changes []string
	for _, t := range add {
		changes = append(changes, "+"+t)
	}
	for _, t := range remove {
		changes = append(changes, "-"+t)
	}
	return strings.Join(changes, " ")
}
