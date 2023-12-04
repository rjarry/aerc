package commands

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib/ui"
	"git.sr.ht/~rjarry/aerc/log"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/go-opt"
)

type Menu struct {
	ErrExit     bool   `opt:"-e"`
	Background  bool   `opt:"-b"`
	Accounts    bool   `opt:"-a"`
	Directories bool   `opt:"-d"`
	Command     string `opt:"-c"`
	Xargs       string `opt:"..." complete:"CompleteXargs"`
}

func init() {
	Register(Menu{})
}

func (Menu) Context() CommandContext {
	return GLOBAL
}

func (Menu) Aliases() []string {
	return []string{"menu"}
}

func (*Menu) CompleteXargs(arg string) []string {
	return FilterList(ActiveCommandNames(), arg, nil)
}

func (m Menu) Execute([]string) error {
	if m.Command == "" {
		m.Command = config.General.DefaultMenuCmd
	}
	if m.Command == "" {
		return errors.New(
			"Either -c <command> or default-menu-cmd is required.")
	}
	if _, _, err := ResolveCommand(m.Xargs, nil, nil); err != nil {
		return err
	}

	lines, err := m.feedLines()
	if err != nil {
		return err
	}

	pick, err := os.CreateTemp("", "aerc-menu-*")
	if err != nil {
		return err
	}

	var proc *exec.Cmd
	if strings.Contains(m.Command, "%f") {
		proc = exec.Command("sh", "-c",
			strings.ReplaceAll(m.Command, "%f", opt.QuoteArg(pick.Name())))
	} else {
		proc = exec.Command("sh", "-c", m.Command+" >&3")
		proc.ExtraFiles = append(proc.ExtraFiles, pick)
	}
	if len(lines) > 0 {
		proc.Stdin = strings.NewReader(strings.Join(lines, "\n"))
	}

	xargs := func(err error) {
		var buf []byte
		if err == nil {
			_, err = pick.Seek(0, io.SeekStart)
		}
		if err == nil {
			buf, err = io.ReadAll(pick)
		}
		pick.Close()
		os.Remove(pick.Name())
		if err != nil {
			app.PushError("command failed: " + err.Error())
			return
		}
		if len(buf) == 0 {
			return
		}
		var cmd Command
		var cmdline string

		for _, line := range strings.Split(string(buf), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			cmdline = m.Xargs + " " + line
			cmdline, cmd, err = ResolveCommand(cmdline, nil, nil)
			if err == nil {
				err = ExecuteCommand(cmd, cmdline)
			}
			if err != nil {
				app.PushError(m.Xargs + ": " + err.Error())
				if m.ErrExit {
					return
				}
			}
		}
	}

	if m.Background {
		go func() {
			defer log.PanicHandler()
			xargs(proc.Run())
		}()
	} else {
		term, err := app.NewTerminal(proc)
		if err != nil {
			return err
		}
		term.Focus(true)
		term.OnClose = func(err error) {
			app.CloseDialog()
			xargs(err)
		}

		title := " :" + strings.TrimLeft(m.Xargs, ": \t") + " ... "

		app.AddDialog(app.NewDialog(
			ui.NewBox(term, title, "", app.SelectedAccountUiConfig()),
			// start pos on screen
			func(h int) int {
				return h / 4
			},
			// dialog height
			func(h int) int {
				return h / 2
			},
		))
	}

	return nil
}

func (m Menu) feedLines() ([]string, error) {
	var lines []string

	switch {
	case m.Accounts && m.Directories:
		for _, a := range app.AccountNames() {
			account, _ := app.Account(a)
			a = opt.QuoteArg(a)
			for _, d := range account.Directories().List() {
				dir := account.Directories().Directory(d)
				if dir != nil && dir.Role != models.QueryRole {
					d = opt.QuoteArg(d)
				}
				lines = append(lines, a+" "+d)
			}
		}

	case m.Accounts:
		for _, account := range app.AccountNames() {
			lines = append(lines, opt.QuoteArg(account))
		}

	case m.Directories:
		account := app.SelectedAccount()
		if account == nil {
			return nil, errors.New("No account selected.")
		}
		for _, d := range account.Directories().List() {
			dir := account.Directories().Directory(d)
			if dir != nil && dir.Role != models.QueryRole {
				d = opt.QuoteArg(d)
			}
			lines = append(lines, d)
		}
	}

	return lines, nil
}
