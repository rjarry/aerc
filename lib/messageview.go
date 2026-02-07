package lib

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/ProtonMail/go-crypto/openpgp"
	_ "github.com/emersion/go-message/charset"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib/crypto"
	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/lib/rfc822"
	"git.sr.ht/~rjarry/aerc/models"
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

	// Fetches the full message
	FetchFull(cb func(io.Reader))

	// Fetches a specific body part for this message
	FetchBodyPart(part []int, cb func(io.Reader))

	MessageDetails() *models.MessageDetails

	// SeenFlagSet returns true if the "seen" flag has been set
	SeenFlagSet() bool

	Close()
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
	return slices.ContainsFunc(info.Parts, usePGP)
}

type MessageStoreView struct {
	messageInfo   *models.MessageInfo
	messageStore  *MessageStore
	message       []byte
	details       *models.MessageDetails
	bodyStructure *models.BodyStructure
	setSeen       bool
	ctx           context.Context
	cancel        context.CancelFunc
}

func NewMessageStoreView(messageInfo *models.MessageInfo, setSeen bool,
	store *MessageStore, pgp crypto.Provider, decryptKeys openpgp.PromptFunction,
	innerCb func(MessageView, error),
) {
	cb := func(msv MessageView, err error) {
		if msv != nil && setSeen && err == nil &&
			!messageInfo.Flags.Has(models.SeenFlag) {
			store.Flag([]models.UID{messageInfo.Uid}, models.SeenFlag, true, nil)
		}
		innerCb(msv, err)
	}

	if messageInfo == nil {
		// Call nils to the callback, the split view will use this to
		// display an empty view
		cb(nil, nil)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())

	msv := &MessageStoreView{
		messageInfo, store,
		nil, nil, messageInfo.BodyStructure,
		setSeen,
		ctx, cancel,
	}

	if usePGP(messageInfo.BodyStructure) {
		msv.FetchFull(func(fm io.Reader) {
			reader := rfc822.NewCRLFReader(fm)
			md, err := pgp.Decrypt(reader, decryptKeys)
			if err != nil {
				cb(nil, err)
				return
			}
			msv.message, err = io.ReadAll(md.Body)
			if err != nil {
				cb(nil, err)
				return
			}
			decrypted, err := rfc822.ReadMessage(bytes.NewBuffer(msv.message))
			if err != nil {
				cb(nil, err)
				return
			}
			bs, err := rfc822.ParseEntityStructure(decrypted)
			if rfc822.IsMultipartError(err) {
				log.Warnf("MessageView: %v", err)
				bs = rfc822.CreateTextPlainBody()
			} else if err != nil {
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
}

func (msv *MessageStoreView) SeenFlagSet() bool {
	return msv.setSeen
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

func (msv *MessageStoreView) FetchFull(cb func(io.Reader)) {
	if msv.message == nil && msv.messageStore != nil {
		msv.messageStore.FetchFull(
			msv.ctx, []models.UID{msv.messageInfo.Uid},
			func(fm *types.FullMessage) {
				cb(fm.Content.Reader)
			})
		return
	}
	cb(bytes.NewReader(msv.message))
}

func (msv *MessageStoreView) FetchBodyPart(part []int, cb func(io.Reader)) {
	// Check if we should inline images for HTML parts
	viewerConfig := config.Viewer().ForEnvelope(msv.messageInfo.Envelope)

	// Wrap the callback to apply HTML transformation if needed
	wrappedCb := cb
	if viewerConfig.HtmlInlineImages && msv.isHTMLPart(part) {
		wrappedCb = func(reader io.Reader) {
			// InlineHTMLImages will call our callback
			// asynchronously after fetching all images
			InlineHTMLImages(reader, msv, cb)
		}
	}

	if msv.message == nil && msv.messageStore != nil {
		msv.messageStore.FetchBodyPart(msv.ctx, msv.messageInfo.Uid, part, wrappedCb)
		return
	}

	buf := bytes.NewBuffer(msv.message)
	msg, err := rfc822.ReadMessage(buf)
	if err != nil {
		panic(err)
	}
	reader, err := rfc822.FetchEntityPartReader(msg, part)
	if err != nil {
		errMsg := fmt.Errorf("Failed to fetch message part: %w", err)
		log.Errorf(errMsg.Error())
		if msv.message != nil {
			log.Warnf("Displaying raw message part")
			reader = bytes.NewReader(msv.message)
		} else {
			reader = strings.NewReader(errMsg.Error())
		}
	}
	wrappedCb(reader)
}

// isHTMLPart returns true if the given part index refers to a text/html part
func (msv *MessageStoreView) isHTMLPart(part []int) bool {
	if msv.bodyStructure == nil {
		return false
	}
	partStruct, err := msv.bodyStructure.PartAtIndex(part)
	if err != nil {
		return false
	}
	return partStruct.FullMIMEType() == "text/html"
}

func (msv *MessageStoreView) Close() {
	if msv.cancel != nil {
		msv.cancel()
	}
}
