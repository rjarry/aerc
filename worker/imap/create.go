package imap

import (
	"git.sr.ht/~rjarry/aerc/worker/types"
)

func (imapw *IMAPWorker) handleCreateDirectory(msg *types.CreateDirectory) error {
	if err := imapw.client.Create(msg.Directory); err != nil && !msg.Quiet {
		return err
	}
	return nil
}
