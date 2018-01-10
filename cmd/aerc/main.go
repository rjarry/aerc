package main

import (
	"fmt"
	"time"

	"git.sr.ht/~sircmpwn/aerc2/config"
	"git.sr.ht/~sircmpwn/aerc2/worker"
	"git.sr.ht/~sircmpwn/aerc2/worker/types"
)

func main() {
	conf, err := config.LoadConfig(nil)
	if err != nil {
		panic(err)
	}
	var workers []worker.Worker
	for _, account := range conf.Accounts {
		work, err := worker.NewWorker(account.Source)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Initializing worker %s\n", account.Name)
		go work.Run()
		work.PostAction(types.Configure{Config: account})
		workers = append(workers, work)
	}
	for {
		activity := false
		for _, worker := range workers {
			if msg := worker.GetMessage(); msg != nil {
				activity = true
				fmt.Printf("<- %T\n", msg)
			}
		}
		if !activity {
			time.Sleep(100 * time.Millisecond)
		}
	}
}
