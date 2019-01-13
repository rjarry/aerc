package imap

import (
	"git.sr.ht/~sircmpwn/aerc2/worker/types"
)

func (imapw *IMAPWorker) handleOpenDirectory(msg *types.OpenDirectory) {
	imapw.worker.Logger.Printf("Opening %s", msg.Directory)
	go func() {
		_, err := imapw.client.Select(msg.Directory, false)
		if err != nil {
			imapw.worker.PostMessage(&types.Error{
				Message: types.RespondTo(msg),
				Error:   err,
			}, nil)
		} else {
			imapw.worker.PostMessage(&types.Done{types.RespondTo(msg)}, nil)
		}
	}()
}
