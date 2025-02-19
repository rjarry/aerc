package commands

import (
	"fmt"
	"time"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/lib/log"
)

type Version struct{}

func init() {
	Register(Version{})
}

func (Version) Description() string {
	return "Display the version of the running aerc instance."
}

func (Version) Context() CommandContext {
	return GLOBAL
}

func (Version) Aliases() []string {
	return []string{"version"}
}

func (p Version) Execute(args []string) error {
	app.PushStatus(fmt.Sprint("aerc "+log.BuildInfo), 20*time.Second)
	return nil
}
