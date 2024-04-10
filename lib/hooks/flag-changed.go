package hooks

import (
	"fmt"

	"git.sr.ht/~rjarry/aerc/config"
)

type FlagChanged struct {
	Account  string
	Folder   string
	FlagName string
}

func (m *FlagChanged) Cmd() string {
	return config.Hooks.FlagChanged
}

func (m *FlagChanged) Env() []string {
	env := []string{
		fmt.Sprintf("AERC_ACCOUNT=%s", m.Account),
		fmt.Sprintf("AERC_FOLDER=%s", m.Folder),
		fmt.Sprintf("AERC_FLAG=%s", m.FlagName),
	}

	return env
}
