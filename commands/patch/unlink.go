package patch

import (
	"fmt"
	"time"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/lib/pama"
)

type Unlink struct {
	Tag string `opt:"tag" required:"false" complete:"Complete" desc:"Project tag name."`
}

func init() {
	register(Unlink{})
}

func (Unlink) Description() string {
	return "Delete all patch tracking data for the specified project."
}

func (Unlink) Context() commands.CommandContext {
	return commands.GLOBAL
}

func (Unlink) Aliases() []string {
	return []string{"unlink"}
}

func (*Unlink) Complete(arg string) []string {
	names, err := pama.New().Names()
	if err != nil {
		log.Errorf("failed to get completion: %v", err)
		return nil
	}
	return commands.FilterList(names, arg, nil)
}

func (d Unlink) Execute(args []string) error {
	m := pama.New()

	name := d.Tag
	if name == "" {
		p, err := m.CurrentProject()
		if err != nil {
			return err
		}
		name = p.Name
	}

	err := m.Unlink(name)
	if err != nil {
		return err
	}

	app.PushStatus(fmt.Sprintf("Project '%s' unlinked.", name),
		10*time.Second)
	return nil
}
