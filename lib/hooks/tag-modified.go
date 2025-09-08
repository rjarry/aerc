package hooks

import (
	"fmt"

	"git.sr.ht/~rjarry/aerc/config"
)

type TagModified struct {
	Account string
	Backend string
	Add     []string
	Remove  []string
	Toggle  []string
}

func (m *TagModified) Cmd() string {
	return config.Hooks().TagModified
}

func (m *TagModified) Env() []string {
	env := []string{
		fmt.Sprintf("AERC_ACCOUNT=%s", m.Account),
		fmt.Sprintf("AERC_TAG_ADDED=%v", m.Add),
		fmt.Sprintf("AERC_TAG_REMOVED=%v", m.Remove),
		fmt.Sprintf("AERC_TAG_TOGGLED=%v", m.Toggle),
	}

	return env
}
