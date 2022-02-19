package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"runtime/debug"
	"sort"
	"strings"
	"time"

	"git.sr.ht/~sircmpwn/getopt"
	"github.com/mattn/go-isatty"

	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/commands/account"
	"git.sr.ht/~rjarry/aerc/commands/compose"
	"git.sr.ht/~rjarry/aerc/commands/msg"
	"git.sr.ht/~rjarry/aerc/commands/msgview"
	"git.sr.ht/~rjarry/aerc/commands/terminal"
	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/lib/templates"
	libui "git.sr.ht/~rjarry/aerc/lib/ui"
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
	cmds := getCommands((*aerc).SelectedTab())
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
	for _, set := range getCommands((*aerc).SelectedTab()) {
		completions = append(completions, set.GetCompletions(aerc, cmd)...)
	}
	sort.Strings(completions)
	return completions
}

// set at build time
var Version string

func usage() {
	log.Fatal("Usage: aerc [-v] [mailto:...]")
}

var termsWithStatusLine = []string{"xterm", "tmux", "screen"}

func setWindowTitle() {
	term := strings.ToLower(os.Getenv("TERM"))
	for _, t := range termsWithStatusLine {
		if strings.Contains(term, t) {
			// TODO: avoid hard coding the list of terminals that
			// have status line support.
			os.Stderr.Write([]byte("\x1b]0;aerc\a"))
			return
		}
	}
}

func main() {
	opts, optind, err := getopt.Getopts(os.Args, "v")
	if err != nil {
		log.Print(err)
		usage()
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
		usage()
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

	var (
		logOut io.Writer
		logger *log.Logger
	)
	if !isatty.IsTerminal(os.Stdout.Fd()) {
		logOut = os.Stdout
	} else {
		logOut = ioutil.Discard
		os.Stdout, _ = os.Open(os.DevNull)
	}
	logger = log.New(logOut, "", log.LstdFlags)
	logger.Println("Starting up aerc")

	conf, err := config.LoadConfigFromFile(nil, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	var (
		aerc *widgets.Aerc
		ui   *libui.UI
	)

	defer PanicTermFix(ui) // recover upon panic and try restoring the pty

	aerc = widgets.NewAerc(conf, logger, func(cmd []string) error {
		return execCommand(aerc, ui, cmd)
	}, func(cmd string) []string {
		return getCompletions(aerc, cmd)
	}, &commands.CmdHistory)

	ui, err = libui.Initialize(aerc)
	if err != nil {
		panic(err)
	}
	defer ui.Close()

	if conf.Ui.MouseEnabled {
		ui.EnableMouse()
	}

	logger.Println("Initializing PGP keyring")
	lib.InitKeyring()
	defer lib.UnlockKeyring()

	logger.Println("Starting Unix server")
	as, err := lib.StartServer(logger)
	if err != nil {
		logger.Printf("Failed to start Unix server: %v (non-fatal)", err)
	} else {
		defer as.Close()
		as.OnMailto = aerc.Mailto
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

//FatalTermFix prints the stacktrace upon panic and tries to recover the term
// not doing that leaves the terminal in a broken state
func PanicTermFix(ui *libui.UI) {
	var err interface{}
	if err = recover(); err == nil {
		return
	}
	debug.PrintStack()
	if ui != nil {
		ui.Close()
	}
	fmt.Fprintf(os.Stderr, "aerc crashed: %v\n", err)
	os.Exit(1)

}
