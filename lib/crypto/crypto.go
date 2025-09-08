package crypto

import (
	"bytes"
	"io"
	"slices"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib/crypto/gpg"
	"git.sr.ht/~rjarry/aerc/lib/crypto/pgp"
	"git.sr.ht/~rjarry/aerc/lib/log"
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

func New() Provider {
	switch config.General().PgpProvider {
	case "auto":
		internal := &pgp.Mail{}
		if internal.KeyringExists() {
			log.Debugf("internal pgp keyring exists")
			return internal
		}
		log.Debugf("no internal pgp keyring, using system gpg")
		fallthrough
	case "gpg":
		return &gpg.Mail{}
	case "internal":
		return &pgp.Mail{}
	default:
		return nil
	}
}

func IsEncrypted(bs *models.BodyStructure) bool {
	if bs == nil {
		return false
	}
	if bs.MIMEType == "application" && bs.MIMESubType == "pgp-encrypted" {
		return true
	}
	return slices.ContainsFunc(bs.Parts, IsEncrypted)
}
