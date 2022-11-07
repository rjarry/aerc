package imap

import (
	"strings"

	"github.com/emersion/go-imap"

	"git.sr.ht/~rjarry/aerc/models"
	"github.com/emersion/go-message/charset"
	"github.com/emersion/go-message/mail"
)

func init() {
	imap.CharsetReader = charset.Reader
}

func toSeqSet(uids []uint32) *imap.SeqSet {
	var set imap.SeqSet
	for _, uid := range uids {
		set.AddNum(uid)
	}
	return &set
}

func translateBodyStructure(bs *imap.BodyStructure) *models.BodyStructure {
	if bs == nil {
		return nil
	}
	var parts []*models.BodyStructure
	for _, part := range bs.Parts {
		parts = append(parts, translateBodyStructure(part))
	}

	// TODO: is that all?

	return &models.BodyStructure{
		MIMEType:          bs.MIMEType,
		MIMESubType:       bs.MIMESubType,
		Params:            bs.Params,
		Description:       bs.Description,
		Encoding:          bs.Encoding,
		Parts:             parts,
		Disposition:       bs.Disposition,
		DispositionParams: bs.DispositionParams,
	}
}

func translateEnvelope(e *imap.Envelope) *models.Envelope {
	if e == nil {
		return nil
	}

	// we strip the msgid of "<>" in order to be more compatible with go-message
	// which wants to handle msgids without the markers
	msgID := strings.TrimSuffix(strings.TrimPrefix(e.MessageId, "<"), ">")
	inReplyTo := strings.TrimSuffix(strings.TrimPrefix(e.InReplyTo, "<"), ">")

	return &models.Envelope{
		Date:      e.Date,
		Subject:   e.Subject,
		From:      translateAddresses(e.From),
		ReplyTo:   translateAddresses(e.ReplyTo),
		To:        translateAddresses(e.To),
		Cc:        translateAddresses(e.Cc),
		Bcc:       translateAddresses(e.Bcc),
		MessageId: msgID,
		InReplyTo: inReplyTo,
	}
}

func translateAddresses(addrs []*imap.Address) []*mail.Address {
	var converted []*mail.Address
	for _, addr := range addrs {
		converted = append(converted, &mail.Address{
			Name:    addr.PersonalName,
			Address: addr.Address(),
		})
	}
	return converted
}

var imapToFlag = map[string]models.Flag{
	imap.SeenFlag:     models.SeenFlag,
	imap.RecentFlag:   models.RecentFlag,
	imap.AnsweredFlag: models.AnsweredFlag,
	imap.DeletedFlag:  models.DeletedFlag,
	imap.FlaggedFlag:  models.FlaggedFlag,
}

var flagToImap = map[models.Flag]string{
	models.SeenFlag:     imap.SeenFlag,
	models.RecentFlag:   imap.RecentFlag,
	models.AnsweredFlag: imap.AnsweredFlag,
	models.DeletedFlag:  imap.DeletedFlag,
	models.FlaggedFlag:  imap.FlaggedFlag,
}

func translateImapFlags(imapFlags []string) []models.Flag {
	var flags []models.Flag
	for _, imapFlag := range imapFlags {
		if flag, ok := imapToFlag[imapFlag]; ok {
			flags = append(flags, flag)
		}
	}
	return flags
}

func translateFlags(flags []models.Flag) []string {
	var imapFlags []string
	for _, flag := range flags {
		if imapFlag, ok := flagToImap[flag]; ok {
			imapFlags = append(imapFlags, imapFlag)
		}
	}
	return imapFlags
}
