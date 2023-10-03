package msg

import (
	"time"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

type Copy struct {
	CreateFolders bool   `opt:"-p"`
	Folder        string `opt:"..." metavar:"<folder>"`
}

func init() {
	register(Copy{})
}

func (Copy) Aliases() []string {
	return []string{"cp", "copy"}
}

func (Copy) Complete(args []string) []string {
	return commands.GetFolders(args)
}

func (c Copy) Execute(args []string) error {
	h := newHelper()
	uids, err := h.markedOrSelectedUids()
	if err != nil {
		return err
	}
	store, err := h.store()
	if err != nil {
		return err
	}
	store.Copy(uids, c.Folder,
		c.CreateFolders, func(
			msg types.WorkerMessage,
		) {
			switch msg := msg.(type) {
			case *types.Done:
				app.PushStatus("Messages copied.", 10*time.Second)
				store.Marker().ClearVisualMark()
			case *types.Error:
				app.PushError(msg.Error.Error())
			}
		})
	return nil
}
