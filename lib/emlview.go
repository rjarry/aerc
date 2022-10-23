package lib

import (
	"bytes"
	"io"

	"github.com/ProtonMail/go-crypto/openpgp"
	_ "github.com/emersion/go-message/charset"

	"git.sr.ht/~rjarry/aerc/lib/crypto"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/lib"
)

// EmlMessage implements the RawMessage interface
type EmlMessage []byte

func (fm *EmlMessage) NewReader() (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(*fm)), nil
}

func (fm *EmlMessage) UID() uint32 {
	return 0xFFFFFFF
}

func (fm *EmlMessage) Labels() ([]string, error) {
	return nil, nil
}

func (fm *EmlMessage) ModelFlags() ([]models.Flag, error) {
	return []models.Flag{models.SeenFlag}, nil
}

// NewEmlMessageView provides a MessageView for a full message that is not
// stored in a message store
func NewEmlMessageView(full []byte, pgp crypto.Provider,
	decryptKeys openpgp.PromptFunction, cb func(MessageView, error),
) {
	eml := EmlMessage(full)
	messageInfo, err := lib.MessageInfo(&eml)
	if err != nil {
		cb(nil, err)
		return
	}
	msv := &MessageStoreView{
		messageInfo:   messageInfo,
		messageStore:  nil,
		message:       full,
		details:       nil,
		bodyStructure: nil,
		setSeen:       false,
	}

	if usePGP(messageInfo.BodyStructure) {
		reader := lib.NewCRLFReader(bytes.NewReader(full))
		md, err := pgp.Decrypt(reader, decryptKeys)
		if err != nil {
			cb(nil, err)
			return
		}
		msv.details = md
		msv.message, err = io.ReadAll(md.Body)
		if err != nil {
			cb(nil, err)
			return
		}
	}
	entity, err := lib.ReadMessage(bytes.NewBuffer(msv.message))
	if err != nil {
		cb(nil, err)
		return
	}
	bs, err := lib.ParseEntityStructure(entity)
	if err != nil {
		cb(nil, err)
		return
	}
	msv.bodyStructure = bs
	cb(msv, nil)
}
