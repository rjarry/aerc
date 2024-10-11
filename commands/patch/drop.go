package patch

import (
	"fmt"
	"time"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/lib/pama"
)

type Drop struct {
	Tag string `opt:"tag" complete:"CompleteTag" desc:"Repository patch tag."`
}

func init() {
	register(Drop{})
}

func (Drop) Description() string {
	return "Drop a patch from the repository."
}

func (Drop) Context() commands.CommandContext {
	return commands.GLOBAL
}

func (Drop) Aliases() []string {
	return []string{"drop"}
}

func (*Drop) CompleteTag(arg string) []string {
	patches, err := pama.New().CurrentPatches()
	if err != nil {
		log.Errorf("failed to get current patches: %v", err)
		return nil
	}
	return commands.FilterList(patches, arg, nil)
}

func (r Drop) Execute(args []string) error {
	patch := r.Tag
	err := pama.New().DropPatch(patch)
	if err != nil {
		return err
	}
	app.PushStatus(fmt.Sprintf("Patch %s has been dropped", patch),
		10*time.Second)
	return nil
}
