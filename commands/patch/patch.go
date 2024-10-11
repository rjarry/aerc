package patch

import (
	"errors"
	"fmt"

	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/go-opt/v2"
)

var subCommands map[string]commands.Command

func register(cmd commands.Command) {
	if subCommands == nil {
		subCommands = make(map[string]commands.Command)
	}
	for _, alias := range cmd.Aliases() {
		if subCommands[alias] != nil {
			panic("duplicate sub command alias: " + alias)
		}
		subCommands[alias] = cmd
	}
}

type Patch struct {
	SubCmd commands.Command `opt:"command" action:"ParseSub" complete:"CompleteSubNames" desc:"Sub command."`
	Args   string           `opt:"..." required:"false" complete:"CompleteSubArgs"`
}

func init() {
	commands.Register(Patch{})
}

func (Patch) Description() string {
	return "Local patch management commands."
}

func (Patch) Context() commands.CommandContext {
	return commands.GLOBAL
}

func (Patch) Aliases() []string {
	return []string{"patch"}
}

func (p *Patch) ParseSub(arg string) error {
	cmd, ok := subCommands[arg]
	if ok {
		context := commands.CurrentContext()
		if cmd.Context()&context != 0 {
			p.SubCmd = cmd
			return nil
		}
	}
	return fmt.Errorf("%s unknown sub-command", arg)
}

func (*Patch) CompleteSubNames(arg string) []string {
	context := commands.CurrentContext()
	options := make([]string, 0, len(subCommands))
	for alias, cmd := range subCommands {
		if cmd.Context()&context != 0 {
			options = append(options, alias)
		}
	}
	return commands.FilterList(options, arg, commands.QuoteSpace)
}

func (p *Patch) CompleteSubArgs(arg string) []string {
	if p.SubCmd == nil {
		return nil
	}
	// prepend arbitrary string to arg to work with sub-commands
	options, _ := commands.GetCompletions(p.SubCmd, opt.LexArgs("a "+arg))
	completions := make([]string, 0, len(options))
	for _, o := range options {
		completions = append(completions, o.Value)
	}
	return completions
}

func (p Patch) Execute(args []string) error {
	if p.SubCmd == nil {
		return errors.New("no subcommand found")
	}
	a := opt.QuoteArgs(args[1:]...)
	return commands.ExecuteCommand(p.SubCmd, a.String())
}
