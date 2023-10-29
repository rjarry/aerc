package hooks

import (
	"fmt"

	"git.sr.ht/~rjarry/aerc/config"
)

type MailAdded struct {
	Account string
	Folder  string
}

func (m *MailAdded) Cmd() string {
	return config.Hooks.MailAdded
}

func (m *MailAdded) Env() []string {
	return []string{
		fmt.Sprintf("AERC_ACCOUNT=%s", m.Account),
		fmt.Sprintf("AERC_FOLDER=%s", m.Folder),
	}
}
