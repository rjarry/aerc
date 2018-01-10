package main

import (
	"fmt"

	"git.sr.ht/~sircmpwn/aerc2/config"
	"git.sr.ht/~sircmpwn/aerc2/worker"
	"git.sr.ht/~sircmpwn/aerc2/worker/types"
)

func main() {
	var (
		conf *config.AercConfig
		err  error
	)
	if conf, err = config.LoadConfig(nil); err != nil {
		panic(err)
	}
	workers := make([]worker.Worker, 0)
	for _, account := range conf.Accounts {
		var work worker.Worker
		if work, err = worker.NewWorker(account.Source); err != nil {
			panic(err)
		}
		fmt.Printf("Initializing worker %s\n", account.Name)
		go work.Run()
		work.PostAction(types.Configure{Config: account})
		workers = append(workers, work)
	}
	for {
		for _, worker := range workers {
			if msg := worker.GetMessage(); msg != nil {
				fmt.Printf("<- %T\n", msg)
			}
		}
	}
}
