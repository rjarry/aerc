package commands

import (
	"bytes"
	"errors"
	"path"
	"reflect"
	"sort"
	"strings"
	"unicode"

	"git.sr.ht/~rjarry/go-opt/v2"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/lib/state"
	"git.sr.ht/~rjarry/aerc/lib/templates"
	"git.sr.ht/~rjarry/aerc/models"
)

type CommandContext uint32

const (
	NONE = 1 << iota
	// available everywhere
	GLOBAL
	// only when a message list is focused
	MESSAGE_LIST
	// only when a message viewer is focused
	MESSAGE_VIEWER
	// only when a message composer editor is focused
	COMPOSE_EDIT
	// only when a message composer review screen is focused
	COMPOSE_REVIEW
	// only when a terminal
	TERMINAL
)

func CurrentContext() CommandContext {
	var context CommandContext = GLOBAL

	switch tab := app.SelectedTabContent().(type) {
	case *app.AccountView:
		context |= MESSAGE_LIST
	case *app.Composer:
		if tab.Bindings() == "compose::review" {
			context |= COMPOSE_REVIEW
		} else {
			context |= COMPOSE_EDIT
		}
	case *app.MessageViewer:
		context |= MESSAGE_VIEWER
	case *app.Terminal:
		context |= TERMINAL
	}

	return context
}

type Command interface {
	Description() string
	Context() CommandContext
	Aliases() []string
	Execute([]string) error
}

var allCommands map[string]Command

func Register(cmd Command) {
	if allCommands == nil {
		allCommands = make(map[string]Command)
	}
	for _, alias := range cmd.Aliases() {
		if allCommands[alias] != nil {
			panic("duplicate command alias: " + alias)
		}
		allCommands[alias] = cmd
	}
}

func ActiveCommands() []Command {
	var cmds []Command
	context := CurrentContext()
	seen := make(map[reflect.Type]bool)

	for _, cmd := range allCommands {
		t := reflect.TypeOf(cmd)
		if seen[t] {
			continue
		}
		seen[t] = true
		if cmd.Context()&context != 0 {
			cmds = append(cmds, cmd)
		}
	}

	return cmds
}

func ActiveCommandNames() []string {
	var names []string
	context := CurrentContext()

	for alias, cmd := range allCommands {
		if cmd.Context()&context != 0 {
			names = append(names, alias)
		}
	}

	return names
}

type NoSuchCommand string

func (err NoSuchCommand) Error() string {
	return "Unknown command " + string(err)
}

// Expand non-ambiguous command abbreviations.
//
//	q  --> quit
//	ar --> archive
//	im --> import-mbox
func ExpandAbbreviations(name string) (string, Command, error) {
	context := CurrentContext()
	name = strings.TrimLeft(name, ": \t")

	cmd, found := allCommands[name]
	if found && cmd.Context()&context != 0 {
		return name, cmd, nil
	}

	var candidate Command
	var candidateName string

	for alias, cmd := range allCommands {
		if cmd.Context()&context == 0 || !strings.HasPrefix(alias, name) {
			continue
		}
		if candidate != nil {
			// We have more than one command partially
			// matching the input.
			return name, nil, NoSuchCommand(name)
		}
		// We have a partial match.
		candidate = cmd
		candidateName = alias
	}

	if candidate == nil {
		return name, nil, NoSuchCommand(name)
	}

	return candidateName, candidate, nil
}

func ResolveCommand(
	cmdline string, acct *config.AccountConfig, msg *models.MessageInfo,
) (string, Command, error) {
	cmdline, err := ExpandTemplates(cmdline, acct, msg)
	if err != nil {
		return "", nil, err
	}
	name, rest, didCut := strings.Cut(cmdline, " ")
	name, cmd, err := ExpandAbbreviations(name)
	if err != nil {
		return "", nil, err
	}
	cmdline = name
	if didCut {
		cmdline += " " + rest
	}
	return cmdline, cmd, nil
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
			nil,
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
) (options []opt.Completion, prefix string) {
	// copy zeroed struct
	tmp := reflect.New(reflect.TypeOf(cmd)).Interface().(Command)
	s, err := args.ArgSafe(0)
	if err != nil {
		log.Errorf("completions error: %v", err)
		return options, prefix
	}
	spec := opt.NewCmdSpec(s, tmp)
	return spec.GetCompletions(args)
}

func GetFolders(arg string) []string {
	acct := app.SelectedAccount()
	if acct == nil {
		return make([]string, 0)
	}
	return FilterList(acct.Directories().List(), arg, nil)
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
	names := make([]string, 0, len(templates))
	for n := range templates {
		names = append(names, n)
	}
	sort.Strings(names)
	return FilterList(names, arg, nil)
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
	return FilterList(acct.Labels(), arg, func(s string) string {
		return opt.QuoteArg(prefix+s) + " "
	})
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
