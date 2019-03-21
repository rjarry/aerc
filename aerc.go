package main

import (
	"io"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/mattn/go-isatty"

	"git.sr.ht/~sircmpwn/aerc2/config"
	"git.sr.ht/~sircmpwn/aerc2/commands"
	"git.sr.ht/~sircmpwn/aerc2/commands/account"
	"git.sr.ht/~sircmpwn/aerc2/commands/terminal"
	libui "git.sr.ht/~sircmpwn/aerc2/lib/ui"
	"git.sr.ht/~sircmpwn/aerc2/widgets"
)

func getCommands(selected libui.Drawable) []*commands.Commands {
	switch selected.(type) {
	case *widgets.AccountView:
		return []*commands.Commands{
			account.AccountCommands,
			commands.GlobalCommands,
		}
	case *widgets.TermHost:
		return []*commands.Commands{
			terminal.TerminalCommands,
			commands.GlobalCommands,
		}
	default:
		return []*commands.Commands{commands.GlobalCommands}
	}
}

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

	conf, err := config.LoadConfig(nil)
	if err != nil {
		panic(err)
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
				if i == len(cmds) - 1 {
					return err
				} else {
					continue
				}
			} else if _, ok := err.(commands.ErrorExit); ok {
				ui.Exit = true
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

	for !ui.Exit {
		if !ui.Tick() {
			// ~60 FPS
			time.Sleep(16 * time.Millisecond)
		}
	}
}
