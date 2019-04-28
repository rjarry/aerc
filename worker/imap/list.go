package imap

import (
	"github.com/emersion/go-imap"

	"git.sr.ht/~sircmpwn/aerc2/worker/types"
)

func (imapw *IMAPWorker) handleListDirectories(msg *types.ListDirectories) {
	mailboxes := make(chan *imap.MailboxInfo)
	imapw.worker.Logger.Println("Listing mailboxes")

	go func() {
		for mbox := range mailboxes {
			imapw.worker.PostMessage(&types.Directory{
				Message:    types.RespondTo(msg),
				Name:       mbox.Name,
				Attributes: mbox.Attributes,
			}, nil)
		}
	}()

	if err := imapw.client.List("", "*", mailboxes); err != nil {
		imapw.worker.PostMessage(&types.Error{
			Message: types.RespondTo(msg),
			Error:   err,
		}, nil)
	} else {
		imapw.worker.PostMessage(
			&types.Done{types.RespondTo(msg)}, nil)
	}
}
