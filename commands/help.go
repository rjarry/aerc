package commands

import (
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strings"

	"git.sr.ht/~rjarry/aerc/app"
)

type Help struct {
	Topic string `opt:"topic" action:"ParseTopic" default:"aerc" complete:"CompleteTopic" desc:"Help topic."`
}

func init() {
	Register(Help{})
}

func (Help) Description() string {
	return "Display one of aerc's man pages in the embedded terminal."
}

func (Help) Context() CommandContext {
	return GLOBAL
}

func (Help) Aliases() []string {
	return []string{"help", "man"}
}

var aproposRe = regexp.MustCompile(`(?m)^aerc(?:-([a-z]+))?\s+\([0-9]\)\s+-\s+(.+)$`)

func getTopics() map[string]string {
	topics := map[string]string{"keys": "Display contextual key bindings."}
	cmd := exec.Command("man", "-k", "aerc")
	out, err := cmd.CombinedOutput()
	if err != nil {
		// mandb not up to date, return "something"
		topics["aerc"] = ""
		topics["accounts"] = ""
		topics["binds"] = ""
		topics["config"] = ""
		topics["imap"] = ""
		topics["jmap"] = ""
		topics["maildir"] = ""
		topics["notmuch"] = ""
		topics["patch"] = ""
		topics["search"] = ""
		topics["sendmail"] = ""
		topics["smtp"] = ""
		topics["stylesets"] = ""
		topics["templates"] = ""
		topics["tutorial"] = ""
		return topics
	}
	for _, match := range aproposRe.FindAllSubmatch(out, -1) {
		name := string(match[1])
		if name == "" {
			name = "aerc"
		}
		desc := strings.ReplaceAll(string(match[2]), " for aerc(1)", "")
		if !strings.HasSuffix(desc, ".") {
			desc += "."
		}
		desc = strings.ToUpper(desc[0:1]) + desc[1:]
		topics[name] = desc
	}
	return topics
}

func (*Help) CompleteTopic(arg string) []string {
	var pages []string
	for name, desc := range getTopics() {
		if desc != "" {
			name += "\n" + desc
		}
		pages = append(pages, name)
	}
	sort.Strings(pages)
	return FilterList(pages, arg, nil)
}

func (h *Help) ParseTopic(arg string) error {
	topics := getTopics()
	if _, ok := topics[arg]; ok {
		if arg != "aerc" {
			arg = "aerc-" + arg
		}
		h.Topic = arg
		return nil
	}
	return fmt.Errorf("unknown topic %q", arg)
}

func (h Help) Execute(args []string) error {
	if h.Topic == "aerc-keys" {
		app.AddDialog(app.DefaultDialog(
			app.NewListBox(
				"Bindings: Press <Esc> or <Enter> to close. "+
					"Start typing to filter bindings.",
				app.HumanReadableBindings(),
				app.SelectedAccountUiConfig(),
				func(_ string) {
					app.CloseDialog()
				},
			),
		))
		return nil
	}
	term := Term{Cmd: []string{"man", h.Topic}}
	return term.Execute(args)
}
