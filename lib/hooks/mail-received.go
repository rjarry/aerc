package hooks

import (
	"fmt"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/models"
)

type MailReceived struct {
	Account string
	Folder  string
	MsgInfo *models.MessageInfo
}

func (m *MailReceived) Cmd() string {
	return config.Hooks.MailReceived
}

func (m *MailReceived) Env() []string {
	from := m.MsgInfo.Envelope.From[0]
	return []string{
		fmt.Sprintf("AERC_ACCOUNT=%s", m.Account),
		fmt.Sprintf("AERC_FOLDER=%s", m.Folder),
		fmt.Sprintf("AERC_FROM_NAME=%s", from.Name),
		fmt.Sprintf("AERC_FROM_ADDRESS=%s", from.Address),
		fmt.Sprintf("AERC_SUBJECT=%s", m.MsgInfo.Envelope.Subject),
	}
}
