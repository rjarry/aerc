package lib

import (
	"bytes"
	"io"
	"io/ioutil"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/emersion/go-message"
	_ "github.com/emersion/go-message/charset"

	"git.sr.ht/~rjarry/aerc/lib/crypto"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/lib"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

// This is an abstraction for viewing a message with semi-transparent PGP
// support.
type MessageView interface {
	// Returns the MessageInfo for this message
	MessageInfo() *models.MessageInfo

	// Returns the BodyStructure for this message
	BodyStructure() *models.BodyStructure

	// Returns the message store that this message was originally sourced from
	Store() *MessageStore

	// Fetches a specific body part for this message
	FetchBodyPart(part []int, cb func(io.Reader))

	MessageDetails() *models.MessageDetails
}

func usePGP(info *models.BodyStructure) bool {
	if info == nil {
		return false
	}
	if info.MIMEType == "application" {
		if info.MIMESubType == "pgp-encrypted" ||
			info.MIMESubType == "pgp-signature" {

			return true
		}
	}
	for _, part := range info.Parts {
		if usePGP(part) {
			return true
		}
	}
	return false
}

type MessageStoreView struct {
	messageInfo   *models.MessageInfo
	messageStore  *MessageStore
	message       []byte
	details       *models.MessageDetails
	bodyStructure *models.BodyStructure
}

func NewMessageStoreView(messageInfo *models.MessageInfo,
	store *MessageStore, pgp crypto.Provider, decryptKeys openpgp.PromptFunction,
	cb func(MessageView, error)) {

	msv := &MessageStoreView{messageInfo, store,
		nil, nil, messageInfo.BodyStructure}

	if usePGP(messageInfo.BodyStructure) {
		store.FetchFull([]uint32{messageInfo.Uid}, func(fm *types.FullMessage) {
			reader := lib.NewCRLFReader(fm.Content.Reader)
			md, err := pgp.Decrypt(reader, decryptKeys)
			if err != nil {
				cb(nil, err)
				return
			}
			msv.message, err = ioutil.ReadAll(md.Body)
			if err != nil {
				cb(nil, err)
				return
			}
			decrypted, err := message.Read(bytes.NewBuffer(msv.message))
			if err != nil {
				cb(nil, err)
				return
			}
			bs, err := lib.ParseEntityStructure(decrypted)
			if err != nil {
				cb(nil, err)
				return
			}
			msv.bodyStructure = bs
			msv.details = md
			cb(msv, nil)
		})
	} else {
		cb(msv, nil)
	}
	store.Flag([]uint32{messageInfo.Uid}, models.SeenFlag, true, nil)
}

func (msv *MessageStoreView) MessageInfo() *models.MessageInfo {
	return msv.messageInfo
}

func (msv *MessageStoreView) BodyStructure() *models.BodyStructure {
	return msv.bodyStructure
}

func (msv *MessageStoreView) Store() *MessageStore {
	return msv.messageStore
}

func (msv *MessageStoreView) MessageDetails() *models.MessageDetails {
	return msv.details
}

func (msv *MessageStoreView) FetchBodyPart(part []int, cb func(io.Reader)) {

	if msv.message == nil {
		msv.messageStore.FetchBodyPart(msv.messageInfo.Uid, part, cb)
		return
	}

	buf := bytes.NewBuffer(msv.message)
	msg, err := message.Read(buf)
	if err != nil {
		panic(err)
	}
	reader, err := lib.FetchEntityPartReader(msg, part)
	if err != nil {
		panic(err)
	}
	cb(reader)
}
