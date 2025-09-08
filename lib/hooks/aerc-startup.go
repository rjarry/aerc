package hooks

import (
	"fmt"
	"os"

	"git.sr.ht/~rjarry/aerc/config"
)

type AercStartup struct {
	Version string
}

func (m *AercStartup) Cmd() string {
	return config.Hooks().AercStartup
}

func (m *AercStartup) Env() []string {
	return []string{
		fmt.Sprintf("AERC_VERSION=%s", m.Version),
		fmt.Sprintf("AERC_BINARY=%s", os.Args[0]),
	}
}
