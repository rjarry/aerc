package patch

import (
	"fmt"
	"time"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/lib/pama"
	"git.sr.ht/~rjarry/aerc/log"
)

type Switch struct {
	Project string `opt:"project" complete:"Complete"`
}

func init() {
	register(Switch{})
}

func (Switch) Context() commands.CommandContext {
	return commands.GLOBAL
}

func (Switch) Aliases() []string {
	return []string{"switch"}
}

func (s Switch) Complete(arg string) []string {
	m := pama.New()
	names, err := m.Names()
	if err != nil {
		log.Errorf("failed to get completion: %v", err)
		return nil
	}
	cur, err := m.CurrentProject()
	if err == nil {
		i := 0
		for ; i < len(names); i++ {
			if cur.Name == names[i] {
				names = append(names[:i], names[i+1:]...)
				break
			}
		}
	}
	return commands.FilterList(names, arg, nil)
}

func (s Switch) Execute(_ []string) error {
	name := s.Project
	err := pama.New().SwitchProject(name)
	if err != nil {
		return err
	}
	app.PushStatus(fmt.Sprintf("Project switched to '%s'", name),
		10*time.Second)
	return nil
}
