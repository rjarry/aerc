package commands

import (
	"bytes"
	"errors"
	"path"
	"reflect"
	"sort"
	"strings"
	"unicode"

	"git.sr.ht/~rjarry/go-opt"

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

func ExecuteCommand(cmd Command, cmdline string) error {
	args := opt.LexArgs(cmdline)
	if args.Count() == 0 {
		return errors.New("No arguments")
	}
	log.Tracef("executing command %s", args.String())
	// copy zeroed struct
	tmp := reflect.New(reflect.TypeOf(cmd)).Interface().(Command)
	if err := opt.ArgsToStruct(args.Clone(), tmp); err != nil {
		return err
	}
	return tmp.Execute(args.Args())
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
func GetCompletions(
	cmd Command, args *opt.Args,
) (options []string, prefix string) {
	// copy zeroed struct
	tmp := reflect.New(reflect.TypeOf(cmd)).Interface().(Command)
	spec := opt.NewCmdSpec(args.Arg(0), tmp)
	return spec.GetCompletions(args)
}

func GetFolders(arg string) []string {
	acct := app.SelectedAccount()
	if acct == nil {
		return make([]string, 0)
	}
	return CompletionFromList(acct.Directories().List(), arg)
}

func GetTemplates(arg string) []string {
	templates := make(map[string]bool)
	for _, dir := range config.Templates.TemplateDirs {
		for _, f := range listDir(dir, false) {
			if !isDir(path.Join(dir, f)) {
				templates[f] = true
			}
		}
	}
	names := make([]string, len(templates))
	for n := range templates {
		names = append(names, n)
	}
	sort.Strings(names)
	return CompletionFromList(names, arg)
}

// CompletionFromList provides a convenience wrapper for commands to use in a
// complete callback. It simply matches the items provided in valid
func CompletionFromList(valid []string, arg string) []string {
	return FilterList(valid, arg, "", "", app.SelectedAccountUiConfig().FuzzyComplete)
}

func GetLabels(arg string) []string {
	acct := app.SelectedAccount()
	if acct == nil {
		return make([]string, 0)
	}
	var prefix string
	if arg != "" {
		// + and - are used to denote tag addition / removal and need to
		// be striped only the last tag should be completed, so that
		// multiple labels can be selected
		switch arg[0] {
		case '+':
			prefix = "+"
		case '-':
			prefix = "-"
		}
		arg = strings.TrimLeft(arg, "+-")
	}
	return FilterList(acct.Labels(), arg, prefix, " ", acct.UiConfig().FuzzyComplete)
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
