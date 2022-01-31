package commands

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/google/shlex"

	"git.sr.ht/~rjarry/aerc/widgets"
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

func (cmds *Commands) ExecuteCommand(aerc *widgets.Aerc, args []string) error {
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
		names := cmds.Names()
		sort.Strings(names)
		return names
	}

	if len(args) > 1 || cmd[len(cmd)-1] == ' ' {
		if cmd, ok := cmds.dict()[args[0]]; ok {
			var completions []string
			if len(args) > 1 {
				completions = cmd.Complete(aerc, args[1:])
			} else {
				completions = cmd.Complete(aerc, []string{})
			}
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

func GetFolders(aerc *widgets.Aerc, args []string) []string {
	out := make([]string, 0)
	if len(args) == 0 {
		return aerc.SelectedAccount().Directories().List()
	}
	for _, dir := range aerc.SelectedAccount().Directories().List() {
		if foundInString(dir, args[0], aerc.SelectedAccount().UiConfig().FuzzyFolderComplete) {
			out = append(out, dir)
		}
	}
	return out
}

// CompletionFromList provides a convenience wrapper for commands to use in the
// Complete function. It simply matches the items provided in valid
func CompletionFromList(valid []string, args []string) []string {
	out := make([]string, 0)
	if len(args) == 0 {
		return valid
	}
	for _, v := range valid {
		if hasCaseSmartPrefix(v, args[0]) {
			out = append(out, v)
		}
	}
	return out
}

func GetLabels(aerc *widgets.Aerc, args []string) []string {
	if len(args) == 0 {
		return aerc.SelectedAccount().Labels()
	}

	// + and - are used to denote tag addition / removal and need to be striped
	// only the last tag should be completed, so that multiple labels can be
	// selected
	last := args[len(args)-1]
	others := strings.Join(args[:len(args)-1], " ")
	var prefix string
	switch last[0] {
	case '+':
		prefix = "+"
	case '-':
		prefix = "-"
	default:
		prefix = ""
	}
	trimmed := strings.TrimLeft(last, "+-")

	out := make([]string, 0)
	for _, label := range aerc.SelectedAccount().Labels() {
		if hasCaseSmartPrefix(label, trimmed) {
			var prev string
			if len(others) > 0 {
				prev = others + " "
			}
			out = append(out, fmt.Sprintf("%v%v%v", prev, prefix, label))
		}
	}
	return out
}

func foundInString(s, substring string, fuzzy bool) bool {
	if fuzzy {
		return caseInsensitiveContains(s, substring)
	} else {
		return hasCaseSmartPrefix(s, substring)
	}
}

// hasCaseSmartPrefix checks whether s starts with prefix, using a case
// sensitive match if and only if prefix contains upper case letters.
func hasCaseSmartPrefix(s, prefix string) bool {
	if hasUpper(prefix) {
		return strings.HasPrefix(s, prefix)
	}
	return strings.HasPrefix(strings.ToLower(s), strings.ToLower(prefix))
}

func caseInsensitiveContains(s, substr string) bool {
	s, substr = strings.ToUpper(s), strings.ToUpper(substr)
	return strings.Contains(s, substr)
}

func hasUpper(s string) bool {
	for _, r := range s {
		if unicode.IsUpper(r) {
			return true
		}
	}
	return false
}
