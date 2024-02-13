package imap

import (
	"strings"

	"github.com/emersion/go-imap"

	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

func (imapw *IMAPWorker) handleListDirectories(msg *types.ListDirectories) {
	mailboxes := make(chan *imap.MailboxInfo)
	imapw.worker.Tracef("Listing mailboxes")
	done := make(chan interface{})

	go func() {
		defer log.PanicHandler()

		for mbox := range mailboxes {
			if !canOpen(mbox) {
				// no need to pass this to handlers if it can't be opened
				continue
			}
			dir := &models.Directory{
				Name: mbox.Name,
			}
			for _, attr := range mbox.Attributes {
				attr = strings.TrimPrefix(attr, "\\")
				attr = strings.ToLower(attr)
				role, ok := models.Roles[attr]
				if !ok {
					continue
				}
				dir.Role = role
			}
			if mbox.Name == "INBOX" {
				dir.Role = models.InboxRole
			}
			imapw.worker.PostMessage(&types.Directory{
				Message: types.RespondTo(msg),
				Dir:     dir,
			}, nil)
		}
		done <- nil
	}()

	switch {
	case imapw.liststatus:
		items := []imap.StatusItem{
			imap.StatusMessages,
			imap.StatusRecent,
			imap.StatusUnseen,
		}
		statuses, err := imapw.client.liststatus.ListStatus(
			"",
			"*",
			items,
			mailboxes,
		)
		if err != nil {
			<-done
			imapw.worker.PostMessage(&types.Error{
				Message: types.RespondTo(msg),
				Error:   err,
			}, nil)
			return

		}
		for _, status := range statuses {
			imapw.worker.PostMessage(&types.DirectoryInfo{
				Info: &models.DirectoryInfo{
					Name:   status.Name,
					Exists: int(status.Messages),
					Recent: int(status.Recent),
					Unseen: int(status.Unseen),
				},
			}, nil)
		}
	default:
		err := imapw.client.List("", "*", mailboxes)
		if err != nil {
			<-done
			imapw.worker.PostMessage(&types.Error{
				Message: types.RespondTo(msg),
				Error:   err,
			}, nil)
			return
		}
	}
	<-done
	imapw.worker.PostMessage(
		&types.Done{Message: types.RespondTo(msg)}, nil)
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

func (imapw *IMAPWorker) handleSearchDirectory(msg *types.SearchDirectory) {
	emitError := func(err error) {
		imapw.worker.PostMessage(&types.Error{
			Message: types.RespondTo(msg),
			Error:   err,
		}, nil)
	}

	imapw.worker.Tracef("Executing search")
	criteria := translateSearch(msg.Criteria)

	if msg.Context.Err() != nil {
		imapw.worker.PostMessage(&types.Cancelled{
			Message: types.RespondTo(msg),
		}, nil)
		return
	}

	uids, err := imapw.client.UidSearch(criteria)
	if err != nil {
		emitError(err)
		return
	}

	if msg.Context.Err() != nil {
		imapw.worker.PostMessage(&types.Cancelled{
			Message: types.RespondTo(msg),
		}, nil)
		return
	}

	imapw.worker.PostMessage(&types.SearchResults{
		Message: types.RespondTo(msg),
		Uids:    uids,
	}, nil)
}
