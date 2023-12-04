package patch

import (
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/lib/pama"
)

type Term struct {
	Cmd []string `opt:"..." required:"false"`
}

func init() {
	register(Term{})
}

func (Term) Context() commands.CommandContext {
	return commands.GLOBAL
}

func (Term) Aliases() []string {
	return []string{"term"}
}

func (t Term) Execute(_ []string) error {
	p, err := pama.New().CurrentProject()
	if err != nil {
		return err
	}
	return commands.TermCoreDirectory(t.Cmd, p.Root)
}
