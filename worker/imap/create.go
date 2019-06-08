package imap

import (
	"strings"

	"git.sr.ht/~sircmpwn/aerc/worker/types"
)

func (imapw *IMAPWorker) handleCreateDirectory(msg *types.CreateDirectory) {
	if err := imapw.client.Create(msg.Directory); err != nil {
		if strings.HasPrefix(err.Error(), "Mailbox already exists") {
			// ignore "already exists" error
			return
		}
		imapw.worker.PostMessage(&types.Error{
			Message: types.RespondTo(msg),
			Error:   err,
		}, nil)
	} else {
		imapw.worker.PostMessage(&types.Done{types.RespondTo(msg)}, nil)
	}
}
