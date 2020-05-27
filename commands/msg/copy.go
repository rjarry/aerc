package msg

import (
	"errors"
	"strings"

	"git.sr.ht/~sircmpwn/getopt"

	"git.sr.ht/~sircmpwn/aerc/commands"
	"git.sr.ht/~sircmpwn/aerc/widgets"
	"git.sr.ht/~sircmpwn/aerc/worker/types"
)

type Copy struct{}

func init() {
	register(Copy{})
}

func (Copy) Aliases() []string {
	return []string{"cp", "copy"}
}

func (Copy) Complete(aerc *widgets.Aerc, args []string) []string {
	return commands.GetFolders(aerc, args)
}

func (Copy) Execute(aerc *widgets.Aerc, args []string) error {
	if len(args) == 1 {
		return errors.New("Usage: cp [-p] <folder>")
	}
	opts, optind, err := getopt.Getopts(args, "p")
	if err != nil {
		return err
	}
	var (
		createParents bool
	)
	for _, opt := range opts {
		switch opt.Option {
		case 'p':
			createParents = true
		}
	}
	h := newHelper(aerc)
	uids, err := h.markedOrSelectedUids()
	if err != nil {
		return err
	}
	store, err := h.store()
	if err != nil {
		return err
	}
	store.Copy(uids, strings.Join(args[optind:], " "),
		createParents, func(
			msg types.WorkerMessage) {

			switch msg := msg.(type) {
			case *types.Done:
				aerc.PushStatus("Messages copied.")
			case *types.Error:
				aerc.PushError(" " + msg.Error.Error())
			}
		})
	return nil
}
