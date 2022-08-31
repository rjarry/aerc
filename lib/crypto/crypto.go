package crypto

import (
	"bytes"
	"io"

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
	Init() error
	Close()
	GetSignerKeyId(string) (string, error)
	GetKeyId(string) (string, error)
	ExportKey(string) (io.Reader, error)
}

func New(s string) Provider {
	switch s {
	case "gpg":
		return &gpg.Mail{}
	default:
		return &pgp.Mail{}
	}
}

func IsEncrypted(bs *models.BodyStructure) bool {
	if bs == nil {
		return false
	}
	if bs.MIMEType == "application" && bs.MIMESubType == "pgp-encrypted" {
		return true
	}
	for _, part := range bs.Parts {
		if IsEncrypted(part) {
			return true
		}
	}
	return false
}
