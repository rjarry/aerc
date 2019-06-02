package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/mattn/go-isatty"

	"git.sr.ht/~sircmpwn/aerc/commands"
	"git.sr.ht/~sircmpwn/aerc/commands/account"
	"git.sr.ht/~sircmpwn/aerc/commands/compose"
	"git.sr.ht/~sircmpwn/aerc/commands/msg"
	"git.sr.ht/~sircmpwn/aerc/commands/msgview"
	"git.sr.ht/~sircmpwn/aerc/commands/terminal"
	"git.sr.ht/~sircmpwn/aerc/config"
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

var (
	Prefix   string
	ShareDir string
)

func main() {
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

	conf, err := config.LoadConfig(nil, ShareDir)
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	var (
		aerc *widgets.Aerc
		ui   *libui.UI
	)
	aerc = widgets.NewAerc(conf, logger, func(cmd string) error {
		cmds := getCommands(aerc.SelectedTab())
		for i, set := range cmds {
			err := set.ExecuteCommand(aerc, cmd)
			if _, ok := err.(commands.NoSuchCommand); ok {
				if i == len(cmds)-1 {
					return err
				} else {
					continue
				}
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
	})

	ui, err = libui.Initialize(conf, aerc)
	if err != nil {
		panic(err)
	}
	defer ui.Close()

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
