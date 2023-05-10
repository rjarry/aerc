package commands

import (
	"bytes"
	"errors"
	"sort"
	"strings"
	"unicode"

	"github.com/google/shlex"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib/state"
	"git.sr.ht/~rjarry/aerc/lib/templates"
	"git.sr.ht/~rjarry/aerc/log"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/widgets"
)

type Command interface {
	Aliases() []string
	Execute(*widgets.Aerc, []string) error
	Complete(*widgets.Aerc, []string) []string
}

type OptionsProvider interface {
	Command
	Options() string
}

type OptionCompleter interface {
	OptionsProvider
	CompleteOption(*widgets.Aerc, rune, string) []string
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

func (cmds *Commands) ByName(name string) Command {
	if cmd, ok := cmds.dict()[name]; ok {
		return cmd
	}
	return nil
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

func templateData(
	aerc *widgets.Aerc,
	cfg *config.AccountConfig,
	msg *models.MessageInfo,
) models.TemplateData {
	var folder *models.Directory

	acct := aerc.SelectedAccount()
	if acct != nil {
		folder = acct.Directories().SelectedDirectory()
	}
	if cfg == nil && acct != nil {
		cfg = acct.AccountConfig()
	}
	if msg == nil && acct != nil {
		msg, _ = acct.SelectedMessage()
	}

	var data state.TemplateData

	data.SetAccount(cfg)
	data.SetFolder(folder)
	data.SetInfo(msg, 0, false)

	return &data
}

func (cmds *Commands) ExecuteCommand(
	aerc *widgets.Aerc,
	args []string,
	account *config.AccountConfig,
	msg *models.MessageInfo,
) error {
	if len(args) == 0 {
		return errors.New("Expected a command.")
	}
	if cmd, ok := cmds.dict()[args[0]]; ok {
		log.Tracef("executing command %v", args)
		var buf bytes.Buffer
		data := templateData(aerc, account, msg)

		processedArgs := make([]string, len(args))
		for i, arg := range args {
			t, err := templates.ParseTemplate(arg, arg)
			if err != nil {
				return err
			}
			err = templates.Render(t, &buf, data)
			if err != nil {
				return err
			}
			arg = buf.String()
			buf.Reset()
			processedArgs[i] = arg
		}

		return cmd.Execute(aerc, processedArgs)
	}
	return NoSuchCommand(args[0])
}

// GetCompletions returns the completion options and the command prefix
func (cmds *Commands) GetCompletions(
	aerc *widgets.Aerc, cmd string,
) (options []string, prefix string) {
	log.Tracef("completing command: %s", cmd)

	// start completion
	args, err := splitCmd(cmd)
	if err != nil {
		return
	}

	// nothing entered, list all commands
	if len(args) == 0 {
		options = cmds.Names()
		sort.Strings(options)
		return
	}

	// complete command name
	spaceTerminated := cmd[len(cmd)-1] == ' '
	if len(args) == 1 && !spaceTerminated {
		for _, n := range cmds.Names() {
			options = append(options, n+" ")
		}
		options = CompletionFromList(aerc, options, args)

		return
	}

	// look for command in dictionary
	c, ok := cmds.dict()[args[0]]
	if !ok {
		return
	}

	// complete options
	var spec string
	if provider, ok := c.(OptionsProvider); ok {
		spec = provider.Options()
	}

	parser, err := newParser(cmd, spec, spaceTerminated)
	if err != nil {
		log.Debugf("completion parser failed: %v", err)
		return
	}

	switch parser.kind {
	case SHORT_OPTION:
		for _, r := range strings.ReplaceAll(spec, ":", "") {
			if strings.ContainsRune(parser.flag, r) {
				continue
			}
			option := string(r)
			if strings.Contains(spec, option+":") {
				option += " "
			}
			options = append(options, option)
		}
		prefix = cmd
	case OPTION_ARGUMENT:
		cmpl, ok := c.(OptionCompleter)
		if !ok {
			return
		}
		stem := cmd
		if parser.arg != "" {
			stem = strings.TrimSuffix(cmd, parser.arg)
		}
		pad := ""
		if !strings.HasSuffix(stem, " ") {
			pad += " "
		}
		s := parser.flag
		r := rune(s[len(s)-1])
		for _, option := range cmpl.CompleteOption(aerc, r, parser.arg) {
			options = append(options, pad+escape(option)+" ")
		}
		prefix = stem
	case OPERAND:
		stem := strings.Join(args[:parser.optind], " ")
		for _, option := range c.Complete(aerc, args[1:]) {
			if strings.Contains(option, "  ") {
				option = escape(option)
			}
			options = append(options, " "+option)
		}
		prefix = stem
	}

	return
}

func GetFolders(aerc *widgets.Aerc, args []string) []string {
	acct := aerc.SelectedAccount()
	if acct == nil {
		return make([]string, 0)
	}
	if len(args) == 0 {
		return acct.Directories().List()
	}
	return FilterList(acct.Directories().List(), args[0], "", acct.UiConfig().FuzzyComplete)
}

// CompletionFromList provides a convenience wrapper for commands to use in the
// Complete function. It simply matches the items provided in valid
func CompletionFromList(aerc *widgets.Aerc, valid []string, args []string) []string {
	if len(args) == 0 {
		return valid
	}
	return FilterList(valid, args[0], "", aerc.SelectedAccountUiConfig().FuzzyComplete)
}

func GetLabels(aerc *widgets.Aerc, args []string) []string {
	acct := aerc.SelectedAccount()
	if acct == nil {
		return make([]string, 0)
	}
	if len(args) == 0 {
		return acct.Labels()
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

	var prev string
	if len(others) > 0 {
		prev = others + " "
	}
	out := FilterList(acct.Labels(), trimmed, prev+prefix, acct.UiConfig().FuzzyComplete)
	return out
}

// hasCaseSmartPrefix checks whether s starts with prefix, using a case
// sensitive match if and only if prefix contains upper case letters.
func hasCaseSmartPrefix(s, prefix string) bool {
	if hasUpper(prefix) {
		return strings.HasPrefix(s, prefix)
	}
	return strings.HasPrefix(strings.ToLower(s), strings.ToLower(prefix))
}

func hasUpper(s string) bool {
	for _, r := range s {
		if unicode.IsUpper(r) {
			return true
		}
	}
	return false
}

// splitCmd splits the command into arguments
func splitCmd(cmd string) ([]string, error) {
	args, err := shlex.Split(cmd)
	if err != nil {
		return nil, err
	}
	return args, nil
}

func escape(s string) string {
	if strings.Contains(s, " ") {
		return strings.ReplaceAll(s, " ", "\\ ")
	}
	return s
}
