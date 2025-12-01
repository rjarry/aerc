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

func toSeqSet(uids []models.UID) *imap.SeqSet {
	set := new(imap.SeqSet)
	for _, uid := range uids {
		set.AddNum(models.UidToUint32(uid))
	}
	return set
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
		ContentID:         bs.Id,
	}
}

func translateEnvelope(e *imap.Envelope) *models.Envelope {
	if e == nil {
		return nil
	}

	return &models.Envelope{
		Date:      e.Date,
		Subject:   e.Subject,
		From:      translateAddresses(e.From),
		ReplyTo:   translateAddresses(e.ReplyTo),
		To:        translateAddresses(e.To),
		Cc:        translateAddresses(e.Cc),
		Bcc:       translateAddresses(e.Bcc),
		MessageId: translateMessageID(e.MessageId),
		InReplyTo: translateMessageID(e.InReplyTo),
	}
}

func translateMessageID(messageID string) string {
	// Strip away unwanted characters, go-message expects the message id
	// without brackets, spaces, tabs and new lines.
	return strings.Trim(messageID, "<> \t\r\n")
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

var imapToFlag = map[string]models.Flags{
	imap.SeenFlag:     models.SeenFlag,
	imap.RecentFlag:   models.RecentFlag,
	imap.AnsweredFlag: models.AnsweredFlag,
	imap.DeletedFlag:  models.DeletedFlag,
	imap.FlaggedFlag:  models.FlaggedFlag,
	imap.DraftFlag:    models.DraftFlag,
}

var flagToImap = map[models.Flags]string{
	models.SeenFlag:     imap.SeenFlag,
	models.RecentFlag:   imap.RecentFlag,
	models.AnsweredFlag: imap.AnsweredFlag,
	models.DeletedFlag:  imap.DeletedFlag,
	models.FlaggedFlag:  imap.FlaggedFlag,
	models.DraftFlag:    imap.DraftFlag,
}

func translateImapFlags(imapFlags []string) (models.Flags, []string) {
	var systemFlags models.Flags
	var keywordFlags []string
	for _, imapFlag := range imapFlags {
		if flag, ok := imapToFlag[imapFlag]; ok {
			systemFlags |= flag
		} else {
			keywordFlags = append(keywordFlags, imapFlag)
		}
	}
	return systemFlags, keywordFlags
}

func translateFlags(flags models.Flags) []string {
	var imapFlags []string
	for flag, imapFlag := range flagToImap {
		if flags.Has(flag) {
			imapFlags = append(imapFlags, imapFlag)
		}
	}
	return imapFlags
}
