package main

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"time"

	"git.sr.ht/~sircmpwn/getopt"
	"github.com/mattn/go-isatty"
	"github.com/xo/terminfo"

	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/commands/account"
	"git.sr.ht/~rjarry/aerc/commands/compose"
	"git.sr.ht/~rjarry/aerc/commands/msg"
	"git.sr.ht/~rjarry/aerc/commands/msgview"
	"git.sr.ht/~rjarry/aerc/commands/terminal"
	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/lib/crypto"
	"git.sr.ht/~rjarry/aerc/lib/templates"
	libui "git.sr.ht/~rjarry/aerc/lib/ui"
	"git.sr.ht/~rjarry/aerc/logging"
	"git.sr.ht/~rjarry/aerc/widgets"
)

func getCommands(selected libui.Drawable) []*commands.Commands {
	switch selected.(type) {
	case *widgets.AccountView:
		return []*commands.Commands{
			account.AccountCommands,
			msg.MessageCommands,
			commands.GlobalCommands,
		}
	case *widgets.Composer:
		return []*commands.Commands{
			compose.ComposeCommands,
			commands.GlobalCommands,
		}
	case *widgets.MessageViewer:
		return []*commands.Commands{
			msgview.MessageViewCommands,
			msg.MessageCommands,
			commands.GlobalCommands,
		}
	case *widgets.Terminal:
		return []*commands.Commands{
			terminal.TerminalCommands,
			commands.GlobalCommands,
		}
	default:
		return []*commands.Commands{commands.GlobalCommands}
	}
}

func execCommand(aerc *widgets.Aerc, ui *libui.UI, cmd []string) error {
	cmds := getCommands(aerc.SelectedTabContent())
	for i, set := range cmds {
		err := set.ExecuteCommand(aerc, cmd)
		if _, ok := err.(commands.NoSuchCommand); ok {
			if i == len(cmds)-1 {
				return err
			}
			continue
		} else if _, ok := err.(commands.ErrorExit); ok {
			ui.Exit()
			return nil
		} else if err != nil {
			return err
		} else {
			break
		}
	}
	return nil
}

func getCompletions(aerc *widgets.Aerc, cmd string) []string {
	var completions []string
	for _, set := range getCommands(aerc.SelectedTabContent()) {
		completions = append(completions, set.GetCompletions(aerc, cmd)...)
	}
	sort.Strings(completions)
	return completions
}

// set at build time
var Version string

func usage(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	fmt.Fprintln(os.Stderr, "usage: aerc [-v] [mailto:...]")
	os.Exit(1)
}

func setWindowTitle() {
	ti, err := terminfo.LoadFromEnv()
	if err != nil {
		return
	}

	if !ti.Has(terminfo.HasStatusLine) {
		return
	}

	buf := new(bytes.Buffer)
	ti.Fprintf(buf, terminfo.ToStatusLine)
	fmt.Fprint(buf, "aerc")
	ti.Fprintf(buf, terminfo.FromStatusLine)
	os.Stderr.Write(buf.Bytes())
}

func main() {
	defer logging.PanicHandler()
	opts, optind, err := getopt.Getopts(os.Args, "v")
	if err != nil {
		usage("error: " + err.Error())
		return
	}
	for _, opt := range opts {
		switch opt.Option {
		case 'v':
			fmt.Println("aerc " + Version)
			return
		}
	}
	retryExec := false
	args := os.Args[optind:]
	if len(args) > 1 {
		usage("error: invalid arguments")
		return
	} else if len(args) == 1 {
		arg := args[0]
		err := lib.ConnectAndExec(arg)
		if err == nil {
			return // other aerc instance takes over
		}
		fmt.Fprintf(os.Stderr, "Failed to communicate to aerc: %v\n", err)
		// continue with setting up a new aerc instance and retry after init
		retryExec = true
	}

	if !isatty.IsTerminal(os.Stdout.Fd()) {
		logging.Init()
	}
	logging.Infof("Starting up")

	conf, err := config.LoadConfigFromFile(nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	var (
		aerc *widgets.Aerc
		ui   *libui.UI
	)

	deferLoop := make(chan struct{})

	c := crypto.New(conf.General.PgpProvider)
	c.Init()
	defer c.Close()

	aerc = widgets.NewAerc(conf, c, func(cmd []string) error {
		return execCommand(aerc, ui, cmd)
	}, func(cmd string) []string {
		return getCompletions(aerc, cmd)
	}, &commands.CmdHistory, deferLoop)

	ui, err = libui.Initialize(aerc)
	if err != nil {
		panic(err)
	}
	defer ui.Close()
	logging.UICleanup = func() {
		ui.Close()
	}
	close(deferLoop)

	if conf.Ui.MouseEnabled {
		ui.EnableMouse()
	}

	as, err := lib.StartServer()
	if err != nil {
		logging.Warnf("Failed to start Unix server: %v", err)
	} else {
		defer as.Close()
		as.OnMailto = aerc.Mailto
		as.OnMbox = aerc.Mbox
	}

	// set the aerc version so that we can use it in the template funcs
	templates.SetVersion(Version)

	if retryExec {
		// retry execution
		arg := args[0]
		err := lib.ConnectAndExec(arg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to communicate to aerc: %v\n", err)
			aerc.CloseBackends()
			return
		}
	}

	if isatty.IsTerminal(os.Stderr.Fd()) {
		setWindowTitle()
	}

	for !ui.ShouldExit() {
		for aerc.Tick() {
			// Continue updating our internal state
		}
		if !ui.Tick() {
			// ~60 FPS
			time.Sleep(16 * time.Millisecond)
		}
	}
	aerc.CloseBackends()
}
