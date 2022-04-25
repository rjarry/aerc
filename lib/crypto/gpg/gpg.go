package gpg

import (
	"bytes"
	"io"
	"log"
	"os/exec"

	"git.sr.ht/~rjarry/aerc/lib/crypto/gpg/gpgbin"
	"git.sr.ht/~rjarry/aerc/models"
	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/emersion/go-message/mail"
)

// Mail satisfies the PGPProvider interface in aerc
type Mail struct {
	logger *log.Logger
}

func (m *Mail) Init(l *log.Logger) error {
	m.logger = l
	_, err := exec.LookPath("gpg")
	return err
}

func (m *Mail) Decrypt(r io.Reader, decryptKeys openpgp.PromptFunction) (*models.MessageDetails, error) {
	gpgReader, err := Read(r)
	if err != nil {
		return nil, err
	}
	md := gpgReader.MessageDetails
	md.SignatureValidity = models.Valid
	if md.SignatureError != "" {
		md.SignatureValidity = handleSignatureError(md.SignatureError)
	}
	return md, nil
}

func (m *Mail) ImportKeys(r io.Reader) error {
	return gpgbin.Import(r)
}

func (m *Mail) Encrypt(buf *bytes.Buffer, rcpts []string, signer string, decryptKeys openpgp.PromptFunction, header *mail.Header) (io.WriteCloser, error) {

	return Encrypt(buf, header.Header.Header, rcpts, signer)
}

func (m *Mail) Sign(buf *bytes.Buffer, signer string, decryptKeys openpgp.PromptFunction, header *mail.Header) (io.WriteCloser, error) {
	return Sign(buf, header.Header.Header, signer)
}

func (m *Mail) Close() {}

func handleSignatureError(e string) models.SignatureValidity {
	if e == "gpg: missing public key" {
		return models.UnknownEntity
	}
	if e == "gpg: header hash does not match actual sig hash" {
		return models.MicalgMismatch
	}
	return models.UnknownValidity
}
