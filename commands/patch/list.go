package patch

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"time"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib/pama"
	"git.sr.ht/~rjarry/aerc/lib/pama/models"
	"git.sr.ht/~rjarry/aerc/lib/ui"
	"git.sr.ht/~rjarry/go-opt"
	"git.sr.ht/~rockorager/vaxis"
)

type List struct {
	All bool `opt:"-a"`
}

func init() {
	register(List{})
}

func (List) Context() commands.CommandContext {
	return commands.GLOBAL
}

func (List) Aliases() []string {
	return []string{"list", "ls"}
}

func (l List) Execute(args []string) error {
	m := pama.New()
	current, err := m.CurrentProject()
	if err != nil {
		return err
	}

	projects := []models.Project{current}
	if l.All {
		projects, err = m.Projects("")
		if err != nil {
			return err
		}
	}

	app.PushStatus(fmt.Sprintf("Current project: %s", current.Name), 30*time.Second)

	createWidget := func(r io.Reader) (ui.DrawableInteractive, error) {
		pagerCmd, err := app.CmdFallbackSearch(config.PagerCmds(), true)
		if err != nil {
			return nil, err
		}

		cmd := opt.SplitArgs(pagerCmd)
		pager := exec.Command(cmd[0], cmd[1:]...)
		pager.Stdin = r

		term, err := app.NewTerminal(pager)
		if err != nil {
			return nil, err
		}
		start := time.Now()
		term.OnClose = func(err error) {
			if time.Since(start) > 250*time.Millisecond {
				app.CloseDialog()
				return
			}
			term.OnEvent = func(_ vaxis.Event) bool {
				app.CloseDialog()
				return true
			}
		}
		return term, nil
	}

	viewer, err := createWidget(m.NewReader(projects))
	if err != nil {
		viewer = app.NewListBox(
			"Press <Esc> or <Enter> to close. "+
				"Start typing to filter.",
			numerify(m.NewReader(projects)), app.SelectedAccountUiConfig(),
			func(_ string) { app.CloseDialog() },
		)
	}

	app.AddDialog(app.LargeDialog(
		ui.NewBox(viewer, "Patch Management", "",
			app.SelectedAccountUiConfig(),
		),
	))

	return nil
}

func numerify(r io.Reader) []string {
	var lines []string
	nr := 1
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		s := scanner.Text()
		lines = append(lines, fmt.Sprintf("%3d %s", nr, s))
		nr++
	}
	return lines
}
