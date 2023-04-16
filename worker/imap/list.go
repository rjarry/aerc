package imap

import (
	"github.com/emersion/go-imap"

	"git.sr.ht/~rjarry/aerc/log"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

func (imapw *IMAPWorker) handleListDirectories(msg *types.ListDirectories) {
	mailboxes := make(chan *imap.MailboxInfo)
	log.Tracef("Listing mailboxes")
	done := make(chan interface{})

	go func() {
		defer log.PanicHandler()

		for mbox := range mailboxes {
			if !canOpen(mbox) {
				// no need to pass this to handlers if it can't be opened
				continue
			}
			imapw.worker.PostMessage(&types.Directory{
				Message: types.RespondTo(msg),
				Dir: &models.Directory{
					Name:       mbox.Name,
					Attributes: mbox.Attributes,
				},
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
					Flags:          status.Flags,
					Name:           status.Name,
					ReadOnly:       status.ReadOnly,
					AccurateCounts: true,

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

func canOpen(mbox *imap.MailboxInfo) bool {
	for _, attr := range mbox.Attributes {
		if attr == imap.NoSelectAttr {
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

	log.Tracef("Executing search")
	criteria, err := parseSearch(msg.Argv)
	if err != nil {
		emitError(err)
		return
	}

	uids, err := imapw.client.UidSearch(criteria)
	if err != nil {
		emitError(err)
		return
	}

	imapw.worker.PostMessage(&types.SearchResults{
		Message: types.RespondTo(msg),
		Uids:    uids,
	}, nil)
}
