package patch

import (
	"fmt"
	"time"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/lib/pama"
	"git.sr.ht/~rjarry/aerc/log"
)

type Delete struct {
	Tag string `opt:"tag" required:"false" complete:"Complete"`
}

func init() {
	register(Delete{})
}

func (Delete) Aliases() []string {
	return []string{"delete"}
}

func (*Delete) Complete(arg string) []string {
	names, err := pama.New().Names()
	if err != nil {
		log.Errorf("failed to get completion: %v", err)
		return nil
	}
	return commands.FilterList(names, arg, nil)
}

func (d Delete) Execute(args []string) error {
	m := pama.New()

	name := d.Tag
	if name == "" {
		p, err := m.CurrentProject()
		if err != nil {
			return err
		}
		name = p.Name
	}

	err := m.Delete(name)
	if err != nil {
		return err
	}

	app.PushStatus(fmt.Sprintf("Project '%s' deleted.", name),
		10*time.Second)
	return nil
}
