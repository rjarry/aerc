package imap

import (
	"fmt"
	"strings"

	"github.com/emersion/go-imap"

	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

func (imapw *IMAPWorker) handleListDirectories(msg *types.ListDirectories) error {
	imapw.worker.Tracef("Listing mailboxes")

	var mailboxes []*imap.MailboxInfo
	var statuses []*imap.MailboxStatus
	var err error

	if imapw.liststatus {
		items := []imap.StatusItem{imap.StatusUidValidity}
		mailboxes, statuses, err = imapw.listMailboxesStatus(items)
	} else {
		mailboxes, err = imapw.listMailboxes()
	}
	if err != nil {
		return err
	}

	statusMap := make(map[string]*imap.MailboxStatus)
	for _, status := range statuses {
		statusMap[status.Name] = status
	}

	labels := make([]string, 0)
	provider := imapw.config.provider
	useLabels := provider == GMail || provider == Proton

	for _, mbox := range mailboxes {
		if !canOpen(mbox) {
			continue
		}
		dir := &models.Directory{
			Name: mbox.Name,
		}
		if status, ok := statusMap[mbox.Name]; ok && status.UidValidity != 0 {
			dir.Uid = fmt.Sprintf("%d", status.UidValidity)
		}
		switch provider {
		case GMail:
			labels = append(labels, mbox.Name)
		case Proton:
			if after, ok := strings.CutPrefix(mbox.Name, "Labels/"); ok {
				labels = append(labels, after)
			}
		}
		for _, attr := range mbox.Attributes {
			attr = strings.TrimPrefix(attr, "\\")
			attr = strings.ToLower(attr)
			if role, ok := models.Roles[attr]; ok {
				dir.Role = role
			}
		}
		if mbox.Name == "INBOX" {
			dir.Role = models.InboxRole
		}
		imapw.worker.PostMessage(&types.Directory{
			Message: types.RespondTo(msg),
			Dir:     dir,
		}, nil)
	}

	if useLabels {
		imapw.worker.Debugf("Available labels: %s", labels)
		imapw.worker.PostMessage(&types.LabelList{Labels: labels}, nil)
	}

	return nil
}

func (imapw *IMAPWorker) listMailboxes() ([]*imap.MailboxInfo, error) {
	ch := make(chan *imap.MailboxInfo)
	done := make(chan []*imap.MailboxInfo)
	go func() {
		defer log.PanicHandler()
		var list []*imap.MailboxInfo
		for mbox := range ch {
			list = append(list, mbox)
		}
		done <- list
	}()
	err := imapw.client.List("", "*", ch)
	return <-done, err
}

func (imapw *IMAPWorker) listMailboxesStatus(
	items []imap.StatusItem,
) ([]*imap.MailboxInfo, []*imap.MailboxStatus, error) {
	ch := make(chan *imap.MailboxInfo)
	done := make(chan []*imap.MailboxInfo)
	go func() {
		defer log.PanicHandler()
		var list []*imap.MailboxInfo
		for mbox := range ch {
			list = append(list, mbox)
		}
		done <- list
	}()
	statuses, err := imapw.client.liststatus.ListStatus("", "*", items, ch)
	return <-done, statuses, err
}

const NonExistentAttr = "\\NonExistent"

func canOpen(mbox *imap.MailboxInfo) bool {
	for _, attr := range mbox.Attributes {
		if attr == imap.NoSelectAttr ||
			attr == NonExistentAttr {
			return false
		}
	}
	return true
}

func (imapw *IMAPWorker) handleSearchDirectory(msg *types.SearchDirectory) error {
	imapw.worker.Tracef("Executing search")

	if msg.Context().Err() != nil {
		return msg.Context().Err()
	}
	if err := imapw.ensureSelected(msg.Directory); err != nil {
		return err
	}

	// Try Gmail X-GM-EXT-1 search first if available
	if imapw.caps.Has("X-GM-EXT-1") && imapw.handleGmailSearch(msg) {
		return nil
	}

	criteria := translateSearch(msg.Criteria)

	uids, err := imapw.client.UidSearch(criteria)
	if err != nil {
		return err
	}

	if msg.Context().Err() != nil {
		return msg.Context().Err()
	}

	imapw.worker.PostMessage(&types.SearchResults{
		Message: types.RespondTo(msg),
		Uids:    imapw.Uint32ToUidList(uids),
	}, nil)

	return nil
}
