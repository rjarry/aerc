package imap

import (
	"github.com/emersion/go-imap"

	"git.sr.ht/~sircmpwn/aerc/models"
	"github.com/emersion/go-message/charset"
)

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

	// we need to decode, because imap store do not use MessageInfo()
	// which do it via go-message
	desc, _ := charset.DecodeHeader(bs.Description)
	params := map[string]string{}
	for k, v := range bs.Params {
		params[k], _ = charset.DecodeHeader(v)
	}
	dispParams := map[string]string{}
	for k, v := range bs.DispositionParams {
		dispParams[k], _ = charset.DecodeHeader(v)
	}

	// TODO: is that all?

	return &models.BodyStructure{
		MIMEType:          bs.MIMEType,
		MIMESubType:       bs.MIMESubType,
		Params:            params,
		Description:       desc,
		Encoding:          bs.Encoding,
		Parts:             parts,
		Disposition:       bs.Disposition,
		DispositionParams: dispParams,
	}
}

func translateEnvelope(e *imap.Envelope) *models.Envelope {
	if e == nil {
		return nil
	}
	// TODO: where we should send error?
	subject, _ := charset.DecodeHeader(e.Subject)

	return &models.Envelope{
		Date:      e.Date,
		Subject:   subject,
		From:      translateAddresses(e.From),
		ReplyTo:   translateAddresses(e.ReplyTo),
		To:        translateAddresses(e.To),
		Cc:        translateAddresses(e.Cc),
		Bcc:       translateAddresses(e.Bcc),
		MessageId: e.MessageId,
	}
}

func translateAddress(a *imap.Address) *models.Address {
	if a == nil {
		return nil
	}
	personalName, _ := charset.DecodeHeader(a.PersonalName)
	return &models.Address{
		Name:    personalName,
		Mailbox: a.MailboxName,
		Host:    a.HostName,
	}
}

func translateAddresses(addrs []*imap.Address) []*models.Address {
	var converted []*models.Address
	for _, addr := range addrs {
		converted = append(converted, translateAddress(addr))
	}
	return converted
}

var flagMap = map[string]models.Flag{
	imap.SeenFlag:     models.SeenFlag,
	imap.RecentFlag:   models.RecentFlag,
	imap.AnsweredFlag: models.AnsweredFlag,
	imap.DeletedFlag:  models.DeletedFlag,
	imap.FlaggedFlag:  models.FlaggedFlag,
}

func translateFlags(imapFlags []string) []models.Flag {
	var flags []models.Flag
	for _, imapFlag := range imapFlags {
		if flag, ok := flagMap[imapFlag]; ok {
			flags = append(flags, flag)
		}
	}
	return flags
}
