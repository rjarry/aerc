package account

import (
	"errors"
	"strconv"
	"strings"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
)

type Split struct {
	Size  int `opt:"n" required:"false" action:"ParseSize"`
	Delta bool
}

func init() {
	commands.Register(Split{})
}

func (Split) Context() commands.CommandContext {
	return commands.ACCOUNT
}

func (s *Split) ParseSize(arg string) error {
	i, err := strconv.ParseInt(arg, 10, 64)
	if err != nil {
		return err
	}
	s.Size = int(i)
	if strings.HasPrefix(arg, "+") || strings.HasPrefix(arg, "-") {
		s.Delta = true
	}
	return nil
}

func (Split) Aliases() []string {
	return []string{"split", "vsplit", "hsplit"}
}

func (s Split) Execute(args []string) error {
	acct := app.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	store := app.SelectedAccount().Store()
	if store == nil {
		return errors.New("Cannot perform action. Messages still loading")
	}

	if s.Size == 0 && acct.SplitSize() == 0 {
		if args[0] == "split" || args[0] == "hsplit" {
			s.Size = app.SelectedAccount().Messages().Height() / 4
		} else {
			s.Size = app.SelectedAccount().Messages().Width() / 2
		}
	}
	if s.Delta {
		acct.SetSplitSize(acct.SplitSize() + s.Size)
		return nil
	}
	if s.Size == acct.SplitSize() {
		// Repeated commands of the same size have the effect of
		// toggling the split
		s.Size = 0
	}
	if s.Size < 0 {
		// Don't allow split to go negative
		s.Size = 1
	}
	switch args[0] {
	case "split", "hsplit":
		return acct.Split(s.Size)
	case "vsplit":
		return acct.Vsplit(s.Size)
	}
	return nil
}
