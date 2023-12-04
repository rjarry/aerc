package msg

import (
	"fmt"
	"time"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

type Copy struct {
	CreateFolders bool   `opt:"-p"`
	Folder        string `opt:"folder" complete:"CompleteFolder"`
}

func init() {
	commands.Register(Copy{})
}

func (Copy) Context() commands.CommandContext {
	return commands.MESSAGE
}

func (Copy) Aliases() []string {
	return []string{"cp", "copy"}
}

func (*Copy) CompleteFolder(arg string) []string {
	return commands.GetFolders(arg)
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
				var s string
				if len(uids) > 1 {
					s = "%d messages copied to %s"
				} else {
					s = "%d message copied to %s"
				}
				app.PushStatus(fmt.Sprintf(s, len(uids), c.Folder), 10*time.Second)
				store.Marker().ClearVisualMark()
			case *types.Error:
				app.PushError(msg.Error.Error())
			}
		})
	return nil
}
