package hooks

import (
	"fmt"

	"git.sr.ht/~rjarry/aerc/config"
	"github.com/emersion/go-message/mail"
)

type MailSent struct {
	Account string
	Backend string
	Header  *mail.Header
}

func (m *MailSent) Cmd() string {
	return config.Hooks().MailSent
}

func (m *MailSent) Env() []string {
	from, _ := mail.ParseAddress(m.Header.Get("From"))
	env := []string{
		fmt.Sprintf("AERC_ACCOUNT=%s", m.Account),
		fmt.Sprintf("AERC_ACCOUNT_BACKEND=%s", m.Backend),
		fmt.Sprintf("AERC_FROM_NAME=%s", from.Name),
		fmt.Sprintf("AERC_FROM_ADDRESS=%s", from.Address),
		fmt.Sprintf("AERC_SUBJECT=%s", m.Header.Get("Subject")),
		fmt.Sprintf("AERC_TO=%s", m.Header.Get("To")),
		fmt.Sprintf("AERC_CC=%s", m.Header.Get("Cc")),
	}

	return env
}
