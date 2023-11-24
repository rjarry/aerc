package patch

import (
	"fmt"
	"os"
	"time"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/lib/pama"
)

type Cd struct{}

func init() {
	register(Cd{})
}

func (Cd) Aliases() []string {
	return []string{"cd"}
}

func (Cd) Execute(args []string) error {
	p, err := pama.New().CurrentProject()
	if err != nil {
		return err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	if cwd == p.Root {
		app.PushStatus("Already here.", 10*time.Second)
		return nil
	}
	err = os.Chdir(p.Root)
	if err != nil {
		return err
	}
	app.PushStatus(fmt.Sprintf("Changed to %s.", p.Root),
		10*time.Second)
	return nil
}
