package imap

import (
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
	"github.com/emersion/go-imap"
)

func (w *IMAPWorker) handleCheckMailMessage(msg *types.CheckMail) {
	items := []imap.StatusItem{
		imap.StatusMessages,
		imap.StatusRecent,
		imap.StatusUnseen,
		imap.StatusUidNext,
	}
	var (
		statuses  []*imap.MailboxStatus
		err       error
		remaining []string
	)
	switch {
	case w.liststatus:
		w.worker.Tracef("Checking mail with LIST-STATUS")
		ref := ""
		if len(msg.Directories) == 1 {
			// If checking a single directory, restrict the ListStatus to it.
			ref = msg.Directories[0]
		}
		statuses, err = w.client.liststatus.ListStatus(ref, "*", items, nil)
		if err == nil && len(statuses) == 0 && len(msg.Directories) == 1 {
			// For providers such as Zoho, we might get an empty list and
			// no error when ref contains the name of a single directory.
			// Workaround this bug by retrying with "".
			statuses, err = w.client.liststatus.ListStatus("", "*", items, nil)
		}
		if err != nil {
			w.worker.PostMessage(&types.Error{
				Message: types.RespondTo(msg),
				Error:   err,
			}, nil)
			return
		}
	default:
		for _, dir := range msg.Directories {
			if len(w.worker.Actions()) > 0 {
				remaining = append(remaining, dir)
				continue
			}
			w.worker.Tracef("Getting status of directory %s", dir)
			status, err := w.client.Status(dir, items)
			if err != nil {
				w.worker.PostMessage(&types.Error{
					Message: types.RespondTo(msg),
					Error:   err,
				}, nil)
				continue
			}
			statuses = append(statuses, status)
		}
	}
	for _, status := range statuses {
		refetch := false
		if status.Name == w.selected.Name {
			if status.UidNext != w.selected.UidNext {
				refetch = true
			} else if status.Messages != w.selected.Messages {
				// Some providers (e.g. O365 upon new mail) might report the
				// same UIDNEXT even though the contents of the folder has
				// changed. So force a refetch if the server reports a
				// number of messages different from what we currently have.
				refetch = true
			}
			w.selected = status
		}
		w.worker.PostMessage(&types.DirectoryInfo{
			Info: &models.DirectoryInfo{
				Name:   status.Name,
				Exists: int(status.Messages),
				Recent: int(status.Recent),
				Unseen: int(status.Unseen),
			},
			Refetch: refetch,
		}, nil)
	}
	if len(remaining) > 0 {
		w.worker.PostMessage(&types.CheckMailDirectories{
			Message:     types.RespondTo(msg),
			Directories: remaining,
		}, nil)
		return
	}
	w.worker.PostMessage(&types.Done{Message: types.RespondTo(msg)}, nil)
}
