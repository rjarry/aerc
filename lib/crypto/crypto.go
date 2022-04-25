package crypto

import (
	"bytes"
	"io"
	"log"

	"git.sr.ht/~rjarry/aerc/lib/crypto/gpg"
	"git.sr.ht/~rjarry/aerc/lib/crypto/pgp"
	"git.sr.ht/~rjarry/aerc/models"
	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/emersion/go-message/mail"
)

type Provider interface {
	Decrypt(io.Reader, openpgp.PromptFunction) (*models.MessageDetails, error)
	Encrypt(*bytes.Buffer, []string, string, openpgp.PromptFunction, *mail.Header) (io.WriteCloser, error)
	Sign(*bytes.Buffer, string, openpgp.PromptFunction, *mail.Header) (io.WriteCloser, error)
	ImportKeys(io.Reader) error
	Init(*log.Logger) error
	Close()
}

func New(s string) Provider {
	switch s {
	case "gpg":
		return &gpg.Mail{}
	default:
		return &pgp.Mail{}
	}
}
