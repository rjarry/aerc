package imap

import (
	"github.com/emersion/go-imap"

	"git.sr.ht/~sircmpwn/aerc2/worker/types"
)

func (imapw *IMAPWorker) handleOpenDirectory(msg *types.OpenDirectory) {
	imapw.worker.Logger.Printf("Opening %s", msg.Directory)

	_, err := imapw.client.Select(msg.Directory, false)
	if err != nil {
		imapw.worker.PostMessage(&types.Error{
			Message: types.RespondTo(msg),
			Error:   err,
		}, nil)
	} else {
		imapw.worker.PostMessage(&types.Done{types.RespondTo(msg)}, nil)
	}
}

func (imapw *IMAPWorker) handleFetchDirectoryContents(
	msg *types.FetchDirectoryContents) {

	imapw.worker.Logger.Printf("Fetching UID list")

	seqSet := &imap.SeqSet{}
	seqSet.AddRange(1, imapw.selected.Messages)
	uids, err := imapw.client.UidSearch(&imap.SearchCriteria{
		SeqNum: seqSet,
	})
	if err != nil {
		imapw.worker.PostMessage(&types.Error{
			Message: types.RespondTo(msg),
			Error:   err,
		}, nil)
	} else {
		imapw.worker.Logger.Printf("Found %d UIDs", len(uids))
		imapw.seqMap = make([]uint32, len(uids))
		imapw.worker.PostMessage(&types.DirectoryContents{
			Message: types.RespondTo(msg),
			Uids:    uids,
		}, nil)
		imapw.worker.PostMessage(&types.Done{types.RespondTo(msg)}, nil)
	}
}
