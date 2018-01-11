package main

import (
	"time"

	"git.sr.ht/~sircmpwn/aerc2/config"
	"git.sr.ht/~sircmpwn/aerc2/ui"
)

func main() {
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
		tab, err := ui.NewAccountTab(&account)
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
