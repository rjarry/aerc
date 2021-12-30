package lib

import (
	"fmt"
	"io"
	"os"
	"path"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
	"github.com/kyoh86/xdg"
)

var (
	Keyring openpgp.EntityList

	locked bool
)

func InitKeyring() {
	os.MkdirAll(path.Join(xdg.DataHome(), "aerc"), 0700)

	lockpath := path.Join(xdg.DataHome(), "aerc", "keyring.lock")
	lockfile, err := os.OpenFile(lockpath, os.O_CREATE|os.O_EXCL, 0600)
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
		return
	} else if err != nil {
		panic(err)
	}
	defer keyfile.Close()

	Keyring, err = openpgp.ReadKeyRing(keyfile)
	if err != nil {
		panic(err)
	}
}

func UnlockKeyring() {
	if !locked {
		return
	}
	lockpath := path.Join(xdg.DataHome(), "aerc", "keyring.lock")
	os.Remove(lockpath)
}

func GetEntityByEmail(email string) (e *openpgp.Entity, err error) {
	for _, entity := range Keyring {
		ident := entity.PrimaryIdentity()
		if ident != nil && ident.UserId.Email == email {
			return entity, nil
		}
	}
	return nil, fmt.Errorf("entity not found in keyring")
}

func GetSignerEntityByEmail(email string) (e *openpgp.Entity, err error) {
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

func ImportKeys(r io.Reader) error {
	keys, err := openpgp.ReadKeyRing(r)
	if err != nil {
		return err
	}
	Keyring = append(Keyring, keys...)
	if locked {
		keypath := path.Join(xdg.DataHome(), "aerc", "keyring.asc")
		keyfile, err := os.OpenFile(keypath, os.O_CREATE|os.O_APPEND, 0600)
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
