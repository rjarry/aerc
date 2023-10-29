package hooks

import (
	"fmt"

	"git.sr.ht/~rjarry/aerc/config"
)

type MailDeleted struct {
	Account string
	Folder  string
}

func (m *MailDeleted) Cmd() string {
	return config.Hooks.MailDeleted
}

func (m *MailDeleted) Env() []string {
	return []string{
		fmt.Sprintf("AERC_ACCOUNT=%s", m.Account),
		fmt.Sprintf("AERC_FOLDER=%s", m.Folder),
	}
}
