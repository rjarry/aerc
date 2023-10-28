package commands

import (
	"bytes"
	"errors"
	"sort"
	"strings"
	"unicode"

	"git.sr.ht/~rjarry/go-opt"
	"github.com/google/shlex"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib/state"
	"git.sr.ht/~rjarry/aerc/lib/templates"
	"git.sr.ht/~rjarry/aerc/log"
	"git.sr.ht/~rjarry/aerc/models"
)

type Command interface {
	Aliases() []string
	Execute([]string) error
	Complete([]string) []string
}

type OptionsProvider interface {
	Command
	Options() string
}

type OptionCompleter interface {
	OptionsProvider
	CompleteOption(rune, string) []string
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
	cfg *config.AccountConfig,
	msg *models.MessageInfo,
) models.TemplateData {
	var folder *models.Directory

	acct := app.SelectedAccount()
	if acct != nil {
		folder = acct.Directories().SelectedDirectory()
	}
	if cfg == nil && acct != nil {
		cfg = acct.AccountConfig()
	}
	if msg == nil && acct != nil {
		msg, _ = acct.SelectedMessage()
	}

	data := state.NewDataSetter()
	data.SetAccount(cfg)
	data.SetFolder(folder)
	data.SetInfo(msg, 0, false)
	if acct != nil {
		acct.SetStatus(func(s *state.AccountState, _ string) {
			data.SetState(s)
		})
	}

	return data.Data()
}

func (cmds *Commands) ExecuteCommand(cmdline string) error {
	args := opt.LexArgs(cmdline)
	name, err := args.ArgSafe(0)
	if err != nil {
		return errors.New("Expected a command after template evaluation.")
	}
	if cmd, ok := cmds.dict()[name]; ok {
		log.Tracef("executing command %s", args.String())
		return cmd.Execute(args.Args())
	}
	return NoSuchCommand(name)
}

// expand template expressions
func ExpandTemplates(
	s string, cfg *config.AccountConfig, msg *models.MessageInfo,
) (string, error) {
	if strings.Contains(s, "{{") && strings.Contains(s, "}}") {
		t, err := templates.ParseTemplate("execute", s)
		if err != nil {
			return "", err
		}

		data := templateData(cfg, msg)

		var buf bytes.Buffer
		err = templates.Render(t, &buf, data)
		if err != nil {
			return "", err
		}

		s = buf.String()
	}

	return s, nil
}

func GetTemplateCompletion(
	cmd string,
) ([]string, string, bool) {
	args, err := splitCmd(cmd)
	if err != nil || len(args) == 0 {
		return nil, "", false
	}

	countLeft := strings.Count(cmd, "{{")
	if countLeft == 0 {
		return nil, "", false
	}
	countRight := strings.Count(cmd, "}}")

	switch {
	case countLeft > countRight:
		// complete template terms
		var i int
		for i = len(cmd) - 1; i >= 0; i-- {
			if strings.ContainsRune("{()| ", rune(cmd[i])) {
				break
			}
		}
		search, prefix := cmd[i+1:], cmd[:i+1]
		padding := strings.Repeat(" ",
			len(search)-len(strings.TrimLeft(search, " ")))
		options := FilterList(
			templates.Terms(),
			strings.TrimSpace(search),
			"",
			app.SelectedAccountUiConfig().FuzzyComplete,
		)
		return options, prefix + padding, true
	case countLeft == countRight:
		// expand template
		s, err := ExpandTemplates(cmd, nil, nil)
		if err != nil {
			log.Warnf("template rendering failed: %v", err)
			return nil, "", false
		}
		return []string{s}, "", true
	}

	return nil, "", false
}

// GetCompletions returns the completion options and the command prefix
func (cmds *Commands) GetCompletions(
	cmd string,
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
		options = CompletionFromList(options, args)

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
		for _, option := range cmpl.CompleteOption(r, parser.arg) {
			options = append(options, pad+escape(option)+" ")
		}
		prefix = stem
	case OPERAND:
		stem := strings.Join(args[:parser.optind], " ")
		for _, option := range c.Complete(args[1:]) {
			if strings.Contains(option, "  ") {
				option = escape(option)
			}
			options = append(options, " "+option)
		}
		prefix = stem
	}

	return
}

func GetFolders(args []string) []string {
	acct := app.SelectedAccount()
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
func CompletionFromList(valid []string, args []string) []string {
	if len(args) == 0 {
		return valid
	}
	return FilterList(valid, args[0], "", app.SelectedAccountUiConfig().FuzzyComplete)
}

func GetLabels(args []string) []string {
	acct := app.SelectedAccount()
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
