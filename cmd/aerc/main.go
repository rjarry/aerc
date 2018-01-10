package main

import (
	"fmt"

	"git.sr.ht/~sircmpwn/aerc2/config"
	"git.sr.ht/~sircmpwn/aerc2/worker"
	"git.sr.ht/~sircmpwn/aerc2/worker/types"
)

func main() {
	var (
		c   *config.AercConfig
		err error
	)
	if c, err = config.LoadConfig(nil); err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", *c)
	w := worker.NewWorker("")
	go w.Run()
	w.PostAction(types.Ping{})
	for {
		if msg := w.GetMessage(); msg != nil {
			fmt.Printf("<- %T: %v\n", msg, msg)
		}
	}
}
