package lib

import (
	"bytes"
	"io"

	"github.com/ProtonMail/go-crypto/openpgp"
	_ "github.com/emersion/go-message/charset"

	"git.sr.ht/~rjarry/aerc/lib/crypto"
	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/lib/rfc822"
	"git.sr.ht/~rjarry/aerc/models"
)

// EmlMessage implements the RawMessage interface
type EmlMessage []byte

func (fm *EmlMessage) NewReader() (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(*fm)), nil
}

func (fm *EmlMessage) UID() models.UID {
	return ""
}

func (fm *EmlMessage) Labels() ([]string, error) {
	return nil, nil
}

func (fm *EmlMessage) ModelFlags() (models.Flags, error) {
	return models.SeenFlag, nil
}

// NewEmlMessageView provides a MessageView for a full message that is not
// stored in a message store
func NewEmlMessageView(full []byte, pgp crypto.Provider,
	decryptKeys openpgp.PromptFunction, cb func(MessageView, error),
) {
	eml := EmlMessage(full)
	messageInfo, err := rfc822.MessageInfo(&eml)
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
		reader := rfc822.NewCRLFReader(bytes.NewReader(full))
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
	entity, err := rfc822.ReadMessage(bytes.NewBuffer(msv.message))
	if err != nil {
		cb(nil, err)
		return
	}
	bs, err := rfc822.ParseEntityStructure(entity)
	if rfc822.IsMultipartError(err) {
		log.Warnf("EmlView: %v", err)
		bs = rfc822.CreateTextPlainBody()
	} else if err != nil {
		cb(nil, err)
		return
	}
	msv.bodyStructure = bs
	cb(msv, nil)
}
