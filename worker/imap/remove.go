package imap

import (
	"git.sr.ht/~rjarry/aerc/worker/types"
)

func (imapw *IMAPWorker) handleRemoveDirectory(msg *types.RemoveDirectory) error {
	if err := imapw.client.Delete(msg.Directory); err != nil && !msg.Quiet {
		return err
	}
	return nil
}
