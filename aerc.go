package main

import (
	"io"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/mattn/go-isatty"

	"git.sr.ht/~sircmpwn/aerc2/config"
	libui "git.sr.ht/~sircmpwn/aerc2/lib/ui"
	"git.sr.ht/~sircmpwn/aerc2/widgets"
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

	conf, err := config.LoadConfig(nil)
	if err != nil {
		panic(err)
	}

	ui, err := libui.Initialize(conf, widgets.NewAerc(conf, logger))
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
