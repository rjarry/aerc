package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"time"

	"git.sr.ht/~sircmpwn/getopt"
	"github.com/mattn/go-isatty"

	"git.sr.ht/~sircmpwn/aerc/commands"
	"git.sr.ht/~sircmpwn/aerc/commands/account"
	"git.sr.ht/~sircmpwn/aerc/commands/compose"
	"git.sr.ht/~sircmpwn/aerc/commands/msg"
	"git.sr.ht/~sircmpwn/aerc/commands/msgview"
	"git.sr.ht/~sircmpwn/aerc/commands/terminal"
	"git.sr.ht/~sircmpwn/aerc/config"
	"git.sr.ht/~sircmpwn/aerc/lib"
	libui "git.sr.ht/~sircmpwn/aerc/lib/ui"
	"git.sr.ht/~sircmpwn/aerc/widgets"
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

func execCommand(aerc *widgets.Aerc, ui *libui.UI, cmd string) error {
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
	cmds := getCommands((*aerc).SelectedTab())
	completions := make([]string, 0)
	for _, set := range cmds {
		opts := set.GetCompletions(aerc, cmd)
		if len(opts) > 0 {
			for _, opt := range opts {
				completions = append(completions, opt)
			}
		}
	}
	return completions
}

var (
	Prefix   string
	ShareDir string
	Version  string
)

func usage() {
	log.Fatal("Usage: aerc [-v] [mailto:...]")
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
	args := os.Args[optind:]
	if len(args) > 1 {
		usage()
		return
	} else if len(args) == 1 {
		lib.ConnectAndExec(args[0])
		return
	}

	var (
		logOut io.Writer
		logger *log.Logger
	)
	if !isatty.IsTerminal(os.Stdout.Fd()) {
		logOut = os.Stdout
	} else {
		logOut = ioutil.Discard
	}
	logger = log.New(logOut, "", log.LstdFlags)
	logger.Println("Starting up aerc")

	conf, err := config.LoadConfigFromFile(nil, ShareDir)
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	var (
		aerc *widgets.Aerc
		ui   *libui.UI
	)

	aerc = widgets.NewAerc(conf, logger, func(cmd string) error {
			return execCommand(aerc, ui, cmd)
		}, func(cmd string) []string {
			return getCompletions(aerc, cmd)
		})

	ui, err = libui.Initialize(conf, aerc)
	if err != nil {
		panic(err)
	}
	defer ui.Close()

	logger.Println("Starting Unix server")
	as, err := lib.StartServer(logger)
	if err != nil {
		logger.Printf("Failed to start Unix server: %v (non-fatal)", err)
	} else {
		defer as.Close()
		as.OnMailto = aerc.Mailto
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
}
