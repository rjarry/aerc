package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/mattn/go-isatty"

	"git.sr.ht/~sircmpwn/aerc2/config"
	"git.sr.ht/~sircmpwn/aerc2/ui"
)

func main() {
	var logOut io.Writer
	var logger *log.Logger
	if !isatty.IsTerminal(os.Stdout.Fd()) {
		logOut = os.Stdout
	} else {
		logOut = ioutil.Discard
	}
	logger = log.New(logOut, "", log.LstdFlags|log.Lshortfile)
	logger.Println("Starting up aerc")

	conf, err := config.LoadConfig(nil)
	if err != nil {
		panic(err)
	}
	_ui, err := ui.Initialize(conf)
	if err != nil {
		panic(err)
	}
	defer _ui.Close()
	for _, account := range conf.Accounts {
		logger.Printf("Initializing account %s\n", account.Name)
		tab, err := ui.NewAccountTab(&account, log.New(
			logOut,
			fmt.Sprintf("[%s] ", account.Name),
			log.LstdFlags|log.Lshortfile))
		if err != nil {
			panic(err)
		}
		_ui.AddTab(tab)
	}
	for !_ui.Exit {
		if !_ui.Tick() {
			time.Sleep(100 * time.Millisecond)
		}
	}
}
