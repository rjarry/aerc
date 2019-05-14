package imap

import (
	"git.sr.ht/~sircmpwn/aerc2/worker/types"
)

func (imapw *IMAPWorker) handleCopyMessages(msg *types.CopyMessages) {
	if err := imapw.client.UidCopy(&msg.Uids, msg.Destination); err != nil {
		imapw.worker.PostMessage(&types.Error{
			Message: types.RespondTo(msg),
			Error:   err,
		}, nil)
	} else {
		imapw.worker.PostMessage(&types.Done{types.RespondTo(msg)}, nil)
	}
}
