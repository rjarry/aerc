package patch

import (
	"fmt"
	"time"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/lib/pama"
	"git.sr.ht/~rjarry/aerc/log"
)

type Remove struct {
	Tag string `opt:"tag" complete:"CompleteTag"`
}

func init() {
	register(Remove{})
}

func (Remove) Context() commands.CommandContext {
	return commands.GLOBAL
}

func (Remove) Aliases() []string {
	return []string{"remove"}
}

func (*Remove) CompleteTag(arg string) []string {
	patches, err := pama.New().CurrentPatches()
	if err != nil {
		log.Errorf("failed to get current patches: %v", err)
		return nil
	}
	return commands.FilterList(patches, arg, nil)
}

func (r Remove) Execute(args []string) error {
	patch := r.Tag
	err := pama.New().RemovePatch(patch)
	if err != nil {
		return err
	}
	app.PushStatus(fmt.Sprintf("Patch %s has been removed", patch),
		10*time.Second)
	return nil
}
