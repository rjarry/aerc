package patch

import (
	"errors"
	"fmt"

	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/go-opt"
)

var (
	PatchCommands *commands.Commands
	subCommands   *commands.Commands
)

func register(cmd commands.Command) {
	if subCommands == nil {
		subCommands = commands.NewCommands()
	}
	subCommands.Register(cmd)
}

func registerPatch(cmd commands.Command) {
	if PatchCommands == nil {
		PatchCommands = commands.NewCommands()
	}
	PatchCommands.Register(cmd)
}

func init() {
	registerPatch(Patch{})
}

type Patch struct {
	SubCmd commands.Command `opt:"command" action:"ParseSub" complete:"CompleteSubNames"`
	Args   string           `opt:"..." required:"false" complete:"CompleteSubArgs"`
}

func (Patch) Aliases() []string {
	return []string{"patch"}
}

func (p *Patch) ParseSub(arg string) error {
	p.SubCmd = subCommands.ByName(arg)
	if p.SubCmd == nil {
		return fmt.Errorf("%s unknown sub-command", arg)
	}
	return nil
}

func (*Patch) CompleteSubNames(arg string) []string {
	options := subCommands.Names()
	return commands.FilterList(options, arg, nil)
}

func (p *Patch) CompleteSubArgs(arg string) []string {
	if p.SubCmd == nil {
		return nil
	}
	// prepend arbitrary string to arg to work with sub-commands
	options, _ := commands.GetCompletions(p.SubCmd, opt.LexArgs("a "+arg))
	return options
}

func (p Patch) Execute(args []string) error {
	if p.SubCmd == nil {
		return errors.New("no subcommand found")
	}
	a := opt.QuoteArgs(args[1:]...)
	return commands.ExecuteCommand(p.SubCmd, a.String())
}
