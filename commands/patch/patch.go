package patch

import (
	"errors"
	"fmt"
	"reflect"
	"sort"

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
	SubCmd commands.Command `opt:":cmd:" action:"ParseSub" complete:"CompleteSubNames"`
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
			// copy zeroed struct
			clone := reflect.New(reflect.TypeOf(cmd)).Interface()
			p.SubCmd = clone.(commands.Command)
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
			options = append(options, alias+"\n"+cmd.Description())
		}
	}
	sort.Strings(options)
	return commands.FilterList(options, arg, nil)
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
	return p.SubCmd.Execute(args)
}
