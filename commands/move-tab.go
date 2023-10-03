package commands

import (
	"strconv"
	"strings"

	"git.sr.ht/~rjarry/aerc/app"
)

type MoveTab struct {
	Index    int `opt:"index" metavar:"[+|-]<index>" action:"ParseIndex"`
	Relative bool
}

func init() {
	register(MoveTab{})
}

func (m *MoveTab) ParseIndex(arg string) error {
	i, err := strconv.ParseInt(arg, 10, 64)
	if err != nil {
		return err
	}
	m.Index = int(i)
	if strings.HasPrefix(arg, "+") || strings.HasPrefix(arg, "-") {
		m.Relative = true
	}
	return nil
}

func (MoveTab) Aliases() []string {
	return []string{"move-tab"}
}

func (MoveTab) Complete(args []string) []string {
	return nil
}

func (m MoveTab) Execute(args []string) error {
	app.MoveTab(m.Index, m.Relative)
	return nil
}
