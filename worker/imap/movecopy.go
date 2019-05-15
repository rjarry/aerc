package imap

import (
	"io"

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

type appendLiteral struct {
	io.Reader
	Length int
}

func (m appendLiteral) Len() int {
	return m.Length
}

func (imapw *IMAPWorker) handleAppendMessage(msg *types.AppendMessage) {
	if err := imapw.client.Append(msg.Destination, msg.Flags, msg.Date,
		&appendLiteral{
			Reader: msg.Reader,
			Length: msg.Length,
		}); err != nil {

		imapw.worker.PostMessage(&types.Error{
			Message: types.RespondTo(msg),
			Error:   err,
		}, nil)
	} else {
		imapw.worker.PostMessage(&types.Done{types.RespondTo(msg)}, nil)
	}
}
