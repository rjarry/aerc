package imap

import (
	"io"

	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

func (imapw *IMAPWorker) handleCopyMessages(msg *types.CopyMessages) error {
	uids := toSeqSet(msg.Uids)
	if err := imapw.client.UidCopy(uids, msg.Destination); err != nil {
		return err
	}
	imapw.worker.PostMessage(&types.MessagesCopied{
		Message:     types.RespondTo(msg),
		Destination: msg.Destination,
		Uids:        msg.Uids,
	}, nil)
	return nil
}

type appendLiteral struct {
	io.Reader
	Length int
}

func (m appendLiteral) Len() int {
	return m.Length
}

func (imapw *IMAPWorker) handleAppendMessage(msg *types.AppendMessage) error {
	if err := imapw.client.Append(msg.Destination, translateFlags(msg.Flags), msg.Date,
		&appendLiteral{
			Reader: msg.Reader,
			Length: msg.Length,
		}); err != nil {
		return err
	}
	return nil
}

func (imapw *IMAPWorker) handleMoveMessages(msg *types.MoveMessages) error {
	drain := imapw.drainUpdates()
	defer drain.Close()

	// Build provider-dependent EXPUNGE handler.
	imapw.BuildExpungeHandler(models.UidToUint32List(msg.Uids), false)

	uids := toSeqSet(msg.Uids)
	if err := imapw.client.UidMove(uids, msg.Destination); err != nil {
		return err
	}
	imapw.worker.PostMessage(&types.MessagesMoved{
		Message:     types.RespondTo(msg),
		Destination: msg.Destination,
		Uids:        msg.Uids,
	}, nil)
	return nil
}
