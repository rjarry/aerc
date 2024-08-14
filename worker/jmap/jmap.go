package jmap

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rockorager/go-jmap"
	"git.sr.ht/~rockorager/go-jmap/mail"
	"git.sr.ht/~rockorager/go-jmap/mail/email"
	"git.sr.ht/~rockorager/go-jmap/mail/mailbox"
	msgmail "github.com/emersion/go-message/mail"
)

func (w *JMAPWorker) translateMsgInfo(m *email.Email) *models.MessageInfo {
	env := &models.Envelope{
		Date:      *m.ReceivedAt,
		Subject:   m.Subject,
		From:      translateAddrList(m.From),
		ReplyTo:   translateAddrList(m.ReplyTo),
		To:        translateAddrList(m.To),
		Cc:        translateAddrList(m.CC),
		Bcc:       translateAddrList(m.BCC),
		MessageId: firstString(m.MessageID),
		InReplyTo: firstString(m.InReplyTo),
	}
	labels := make([]string, 0, len(m.MailboxIDs))
	for id := range m.MailboxIDs {
		if dir, ok := w.mbox2dir[id]; ok {
			labels = append(labels, dir)
		}
	}
	sort.Strings(labels)

	return &models.MessageInfo{
		Envelope:      env,
		Flags:         keywordsToFlags(m.Keywords),
		Uid:           models.UID(m.ID),
		BodyStructure: translateBodyStructure(m.BodyStructure),
		RFC822Headers: translateJMAPHeader(m.Headers),
		Refs:          m.References,
		Labels:        labels,
		Size:          uint32(m.Size),
		InternalDate:  *m.ReceivedAt,
	}
}

func translateJMAPHeader(headers []*email.Header) *msgmail.Header {
	hdr := new(msgmail.Header)
	for _, h := range headers {
		raw := fmt.Sprintf("%s:%s\r\n", h.Name, h.Value)
		hdr.AddRaw([]byte(raw))
	}
	return hdr
}

func flagsToKeywords(flags models.Flags) map[string]bool {
	kw := make(map[string]bool)
	if flags.Has(models.SeenFlag) {
		kw["$seen"] = true
	}
	if flags.Has(models.AnsweredFlag) {
		kw["$answered"] = true
	}
	if flags.Has(models.FlaggedFlag) {
		kw["$flagged"] = true
	}
	if flags.Has(models.DraftFlag) {
		kw["$draft"] = true
	}
	return kw
}

func keywordsToFlags(kw map[string]bool) models.Flags {
	var f models.Flags
	for k, v := range kw {
		if v {
			switch k {
			case "$seen":
				f |= models.SeenFlag
			case "$answered":
				f |= models.AnsweredFlag
			case "$flagged":
				f |= models.FlaggedFlag
			case "$draft":
				f |= models.DraftFlag
			}
		}
	}
	return f
}

func (w *JMAPWorker) MailboxPath(mbox *mailbox.Mailbox) string {
	if mbox == nil {
		return ""
	}
	if mbox.ParentID == "" {
		return mbox.Name
	}
	parent, err := w.cache.GetMailbox(mbox.ParentID)
	if err != nil {
		w.w.Warnf("MailboxPath/GetMailbox: %s", err)
		return mbox.Name
	}
	return w.MailboxPath(parent) + "/" + mbox.Name
}

var jmapRole2aerc = map[mailbox.Role]models.Role{
	mailbox.RoleAll:     models.AllRole,
	mailbox.RoleArchive: models.ArchiveRole,
	mailbox.RoleDrafts:  models.DraftsRole,
	mailbox.RoleInbox:   models.InboxRole,
	mailbox.RoleJunk:    models.JunkRole,
	mailbox.RoleSent:    models.SentRole,
	mailbox.RoleTrash:   models.TrashRole,
}

func firstString(s []string) string {
	if len(s) == 0 {
		return ""
	}
	return s[0]
}

func translateAddrList(addrs []*mail.Address) []*msgmail.Address {
	res := make([]*msgmail.Address, 0, len(addrs))
	for _, a := range addrs {
		res = append(res, &msgmail.Address{Name: a.Name, Address: a.Email})
	}
	return res
}

func translateBodyStructure(part *email.BodyPart) *models.BodyStructure {
	bs := &models.BodyStructure{
		Description: part.Name,
		Encoding:    part.Charset,
		Params: map[string]string{
			"name":    part.Name,
			"charset": part.Charset,
		},
		Disposition: part.Disposition,
		DispositionParams: map[string]string{
			"filename": part.Name,
		},
	}
	bs.MIMEType, bs.MIMESubType, _ = strings.Cut(part.Type, "/")
	for _, sub := range part.SubParts {
		bs.Parts = append(bs.Parts, translateBodyStructure(sub))
	}
	return bs
}

func wrapSetError(err *jmap.SetError) error {
	var s string
	if err.Description != nil {
		s = *err.Description
	} else {
		s = err.Type
		if err.Properties != nil {
			s += fmt.Sprintf(" %v", *err.Properties)
		}
		if s == "invalidProperties: [mailboxIds]" {
			s = "a message must belong to one or more mailboxes"
		}
	}
	return errors.New(s)
}

func wrapMethodError(err *jmap.MethodError) error {
	var s string
	if err.Description != nil {
		s = *err.Description
	} else {
		s = err.Type
	}
	return errors.New(s)
}
