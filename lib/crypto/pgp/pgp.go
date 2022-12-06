package pgp

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"git.sr.ht/~rjarry/aerc/log"
	"git.sr.ht/~rjarry/aerc/models"
	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
	"github.com/emersion/go-message/mail"
	"github.com/emersion/go-pgpmail"
	"github.com/kyoh86/xdg"
	"github.com/pkg/errors"
)

type Mail struct{}

var (
	Keyring openpgp.EntityList

	locked bool
)

func (m *Mail) KeyringExists() bool {
	keypath := path.Join(xdg.DataHome(), "aerc", "keyring.asc")
	keyfile, err := os.Open(keypath)
	if err != nil {
		return false
	}
	defer keyfile.Close()
	_, err = openpgp.ReadKeyRing(keyfile)
	return err == nil
}

func (m *Mail) Init() error {
	log.Debugf("Initializing PGP keyring")
	err := os.MkdirAll(path.Join(xdg.DataHome(), "aerc"), 0o700)
	if err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	lockpath := path.Join(xdg.DataHome(), "aerc", "keyring.lock")
	lockfile, err := os.OpenFile(lockpath, os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		// TODO: Consider connecting to main process over IPC socket
		locked = false
	} else {
		locked = true
		lockfile.Close()
	}

	keypath := path.Join(xdg.DataHome(), "aerc", "keyring.asc")
	keyfile, err := os.Open(keypath)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	defer keyfile.Close()

	Keyring, err = openpgp.ReadKeyRing(keyfile)
	if err != nil {
		return err
	}
	return nil
}

func (m *Mail) Close() {
	if !locked {
		return
	}
	lockpath := path.Join(xdg.DataHome(), "aerc", "keyring.lock")
	os.Remove(lockpath)
}

func (m *Mail) getEntityByEmail(email string) (e *openpgp.Entity, err error) {
	for _, entity := range Keyring {
		ident := entity.PrimaryIdentity()
		if ident != nil && ident.UserId.Email == email {
			return entity, nil
		}
	}
	return nil, fmt.Errorf("entity not found in keyring")
}

func (m *Mail) getSignerEntityByKeyId(id string) (*openpgp.Entity, error) {
	id = strings.ToUpper(id)
	for _, key := range Keyring.DecryptionKeys() {
		if key.Entity == nil {
			continue
		}
		kId := key.Entity.PrimaryKey.KeyIdString()
		if strings.Contains(kId, id) {
			return key.Entity, nil
		}
	}
	return nil, fmt.Errorf("entity not found in keyring")
}

func (m *Mail) getSignerEntityByEmail(email string) (e *openpgp.Entity, err error) {
	for _, key := range Keyring.DecryptionKeys() {
		if key.Entity == nil {
			continue
		}
		ident := key.Entity.PrimaryIdentity()
		if ident != nil && ident.UserId.Email == email {
			return key.Entity, nil
		}
	}
	return nil, fmt.Errorf("entity not found in keyring")
}

func (m *Mail) Decrypt(r io.Reader, decryptKeys openpgp.PromptFunction) (*models.MessageDetails, error) {
	md := new(models.MessageDetails)

	pgpReader, err := pgpmail.Read(r, Keyring, decryptKeys, nil)
	if err != nil {
		return nil, err
	}
	if pgpReader.MessageDetails.IsEncrypted {
		md.IsEncrypted = true
		md.DecryptedWith = pgpReader.MessageDetails.DecryptedWith.Entity.PrimaryIdentity().Name
		md.DecryptedWithKeyId = pgpReader.MessageDetails.DecryptedWith.PublicKey.KeyId
	}
	if pgpReader.MessageDetails.IsSigned {
		// we should consume the UnverifiedBody until EOF in order
		// to get the correct signature data
		data, err := io.ReadAll(pgpReader.MessageDetails.UnverifiedBody)
		if err != nil {
			return nil, err
		}
		pgpReader.MessageDetails.UnverifiedBody = bytes.NewReader(data)

		md.IsSigned = true
		md.SignedBy = ""
		md.SignedByKeyId = pgpReader.MessageDetails.SignedByKeyId
		md.SignatureValidity = models.Valid
		if pgpReader.MessageDetails.SignatureError != nil {
			md.SignatureError = pgpReader.MessageDetails.SignatureError.Error()
			md.SignatureValidity = handleSignatureError(md.SignatureError)
		}
		if pgpReader.MessageDetails.SignedBy != nil {
			md.SignedBy = pgpReader.MessageDetails.SignedBy.Entity.PrimaryIdentity().Name
		}
	}
	md.Body = pgpReader.MessageDetails.UnverifiedBody
	return md, nil
}

func (m *Mail) ImportKeys(r io.Reader) error {
	keys, err := openpgp.ReadKeyRing(r)
	if err != nil {
		return err
	}
	Keyring = append(Keyring, keys...)
	if locked {
		keypath := path.Join(xdg.DataHome(), "aerc", "keyring.asc")
		keyfile, err := os.OpenFile(keypath, os.O_CREATE|os.O_APPEND, 0o600)
		if err != nil {
			return err
		}
		defer keyfile.Close()

		for _, key := range keys {
			if key.PrivateKey != nil {
				err = key.SerializePrivate(keyfile, &packet.Config{})
			} else {
				err = key.Serialize(keyfile)
			}
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *Mail) Encrypt(buf *bytes.Buffer, rcpts []string, signer string, decryptKeys openpgp.PromptFunction, header *mail.Header) (io.WriteCloser, error) {
	var err error
	var to []*openpgp.Entity
	var signerEntity *openpgp.Entity
	if signer != "" {
		signerEntity, err = m.getSigner(signer, decryptKeys)
		if err != nil {
			return nil, err
		}
	}

	for _, rcpt := range rcpts {
		toEntity, err := m.getEntityByEmail(rcpt)
		if err != nil {
			return nil, errors.Wrap(err, "no key for "+rcpt)
		}
		to = append(to, toEntity)
	}

	cleartext, err := pgpmail.Encrypt(buf, header.Header.Header,
		to, signerEntity, nil)
	if err != nil {
		return nil, err
	}
	return cleartext, nil
}

func (m *Mail) Sign(buf *bytes.Buffer, signer string, decryptKeys openpgp.PromptFunction, header *mail.Header) (io.WriteCloser, error) {
	var err error
	var signerEntity *openpgp.Entity
	if signer != "" {
		signerEntity, err = m.getSigner(signer, decryptKeys)
		if err != nil {
			return nil, err
		}
	}
	cleartext, err := pgpmail.Sign(buf, header.Header.Header, signerEntity, nil)
	if err != nil {
		return nil, err
	}
	return cleartext, nil
}

func (m *Mail) getSigner(signer string, decryptKeys openpgp.PromptFunction) (signerEntity *openpgp.Entity, err error) {
	switch strings.Contains(signer, "@") {
	case true:
		signerEntity, err = m.getSignerEntityByEmail(signer)
		if err != nil {
			return nil, err
		}
	case false:
		signerEntity, err = m.getSignerEntityByKeyId(signer)
		if err != nil {
			return nil, err
		}
	}

	key, ok := signerEntity.SigningKey(time.Now())
	if !ok {
		return nil, fmt.Errorf("no signing key found for %s", signer)
	}

	if !key.PrivateKey.Encrypted {
		return signerEntity, nil
	}

	_, err = decryptKeys([]openpgp.Key{key}, false)
	if err != nil {
		return nil, err
	}

	return signerEntity, nil
}

func (m *Mail) GetSignerKeyId(s string) (string, error) {
	var err error
	var signerEntity *openpgp.Entity
	switch strings.Contains(s, "@") {
	case true:
		signerEntity, err = m.getSignerEntityByEmail(s)
		if err != nil {
			return "", err
		}
	case false:
		signerEntity, err = m.getSignerEntityByKeyId(s)
		if err != nil {
			return "", err
		}
	}
	return signerEntity.PrimaryKey.KeyIdString(), nil
}

func (m *Mail) GetKeyId(s string) (string, error) {
	entity, err := m.getEntityByEmail(s)
	if err != nil {
		return "", err
	}
	return entity.PrimaryKey.KeyIdString(), nil
}

func (m *Mail) ExportKey(k string) (io.Reader, error) {
	var err error
	var entity *openpgp.Entity
	switch strings.Contains(k, "@") {
	case true:
		entity, err = m.getSignerEntityByEmail(k)
		if err != nil {
			return nil, err
		}
	case false:
		entity, err = m.getSignerEntityByKeyId(k)
		if err != nil {
			return nil, err
		}
	}
	pks := bytes.NewBuffer(nil)
	err = entity.Serialize(pks)
	if err != nil {
		return nil, fmt.Errorf("pgp: error exporting key: %w", err)
	}
	pka := bytes.NewBuffer(nil)
	w, err := armor.Encode(pka, "PGP PUBLIC KEY BLOCK", map[string]string{})
	if err != nil {
		return nil, fmt.Errorf("pgp: error exporting key: %w", err)
	}
	_, err = w.Write(pks.Bytes())
	if err != nil {
		return nil, fmt.Errorf("pgp: error exporting key: %w", err)
	}
	w.Close()
	return pka, nil
}

func handleSignatureError(e string) models.SignatureValidity {
	if e == "openpgp: signature made by unknown entity" {
		return models.UnknownEntity
	}
	if strings.HasPrefix(e, "pgpmail: unsupported micalg") {
		return models.UnsupportedMicalg
	}
	if strings.HasPrefix(e, "pgpmail") {
		return models.InvalidSignature
	}
	return models.UnknownValidity
}
