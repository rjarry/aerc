package hooks

import (
	"fmt"

	"git.sr.ht/~rjarry/aerc/config"
)

type FlagChanged struct {
	Account  string
	Backend  string
	Folder   string
	Role     string
	FlagName string
}

func (m *FlagChanged) Cmd() string {
	return config.Hooks().FlagChanged
}

func (m *FlagChanged) Env() []string {
	env := []string{
		fmt.Sprintf("AERC_ACCOUNT=%s", m.Account),
		fmt.Sprintf("AERC_ACCOUNT_BACKEND=%s", m.Backend),
		fmt.Sprintf("AERC_FOLDER=%s", m.Folder),
		fmt.Sprintf("AERC_FOLDER_ROLE=%s", m.Role),
		fmt.Sprintf("AERC_FLAG=%s", m.FlagName),
	}

	return env
}
