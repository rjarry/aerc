package main

import (
	"time"

	"git.sr.ht/~sircmpwn/aerc2/config"
	"git.sr.ht/~sircmpwn/aerc2/ui"
	"git.sr.ht/~sircmpwn/aerc2/worker"
	"git.sr.ht/~sircmpwn/aerc2/worker/types"
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
	var workers []worker.Worker
	for _, account := range conf.Accounts {
		work, err := worker.NewWorker(account.Source)
		if err != nil {
			panic(err)
		}
		go work.Run()
		work.PostAction(types.Configure{Config: account})
		workers = append(workers, work)
		_ui.AddTab(ui.NewAccountTab(&account, &work))
	}
	for !_ui.Exit {
		activity := false
		for _, worker := range workers {
			if msg := worker.GetMessage(); msg != nil {
				activity = true
			}
		}
		activity = _ui.Tick() || activity
		if !activity {
			time.Sleep(100 * time.Millisecond)
		}
	}
}
