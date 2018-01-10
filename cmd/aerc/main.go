package main

import (
	"fmt"

	"git.sr.ht/~sircmpwn/aerc2/config"
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
}
