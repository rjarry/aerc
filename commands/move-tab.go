package commands

import (
	"fmt"
	"strconv"
	"strings"

	"git.sr.ht/~sircmpwn/aerc/widgets"
)

type MoveTab struct{}

func init() {
	register(MoveTab{})
}

func (MoveTab) Aliases() []string {
	return []string{"move-tab"}
}

func (MoveTab) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (MoveTab) Execute(aerc *widgets.Aerc, args []string) error {
	if len(args) == 1 {
		return fmt.Errorf("Usage: %s [+|-]<index>", args[0])
	}

	joinedArgs := strings.Join(args[1:], "")

	n, err := strconv.Atoi(joinedArgs)
	if err != nil {
		return fmt.Errorf("failed to parse index argument: %v", err)
	}

	i := aerc.SelectedTabIndex()
	l := aerc.NumTabs()

	if strings.HasPrefix(joinedArgs, "+") {
		i = (i + n) % l
	} else if strings.HasPrefix(joinedArgs, "-") {
		i = (((i + n) % l) + l) % l
	} else {
		i = n
	}

	aerc.MoveTab(i)

	return nil
}
