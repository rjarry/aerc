package commands

import (
	"errors"
	"strings"
	"unicode"

	"github.com/google/shlex"

	"git.sr.ht/~sircmpwn/aerc/widgets"
)

type Command interface {
	Aliases() []string
	Execute(*widgets.Aerc, []string) error
	Complete(*widgets.Aerc, []string) []string
}

type Commands map[string]Command

func NewCommands() *Commands {
	cmds := Commands(make(map[string]Command))
	return &cmds
}

func (cmds *Commands) dict() map[string]Command {
	return map[string]Command(*cmds)
}

func (cmds *Commands) Names() []string {
	names := make([]string, 0)

	for k := range cmds.dict() {
		names = append(names, k)
	}
	return names
}

func (cmds *Commands) Register(cmd Command) {
	// TODO enforce unique aliases, until then, duplicate each
	if len(cmd.Aliases()) < 1 {
		return
	}
	for _, alias := range cmd.Aliases() {
		cmds.dict()[alias] = cmd
	}
}

type NoSuchCommand string

func (err NoSuchCommand) Error() string {
	return "Unknown command " + string(err)
}

type CommandSource interface {
	Commands() *Commands
}

func (cmds *Commands) ExecuteCommand(aerc *widgets.Aerc, cmd string) error {
	args, err := shlex.Split(cmd)
	if err != nil {
		return err
	}
	if len(args) == 0 {
		return errors.New("Expected a command.")
	}
	if cmd, ok := cmds.dict()[args[0]]; ok {
		return cmd.Execute(aerc, args)
	}
	return NoSuchCommand(args[0])
}

func (cmds *Commands) GetCompletions(aerc *widgets.Aerc, cmd string) []string {
	args, err := shlex.Split(cmd)
	if err != nil {
		return nil
	}

	if len(args) == 0 {
		return nil
	}

	if len(args) > 1 {
		if cmd, ok := cmds.dict()[args[0]]; ok {
			completions := cmd.Complete(aerc, args[1:])
			if completions != nil && len(completions) == 0 {
				return nil
			}

			options := make([]string, 0)
			for _, option := range completions {
				options = append(options, args[0]+" "+option)
			}
			return options
		}
		return nil
	}

	names := cmds.Names()
	options := make([]string, 0)
	for _, name := range names {
		if strings.HasPrefix(name, args[0]) {
			options = append(options, name)
		}
	}

	if len(options) > 0 {
		return options
	}
	return nil
}

const caps string = "ABCDEFGHIJKLMNOPQRSTUVXYZ"

func GetFolders(aerc *widgets.Aerc, args []string) []string {
	out := make([]string, 0)
	lower_only := false
	for _, rune := range args[0] {
		lower_only = lower_only || unicode.IsLower(rune)
	}

	for _, dir := range aerc.SelectedAccount().Directories().List() {
		test := dir
		if lower_only {
			test = strings.ToLower(dir)
		}

		if strings.HasPrefix(test, args[0]) {
			out = append(out, dir)
		}
	}
	return out
}
