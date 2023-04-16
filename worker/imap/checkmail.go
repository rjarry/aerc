package imap

import (
	"git.sr.ht/~rjarry/aerc/log"
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
		log.Tracef("Checking mail with LIST-STATUS")
		statuses, err = w.client.liststatus.ListStatus("", "*", items, nil)
		if err != nil {
			w.worker.PostMessage(&types.Error{
				Message: types.RespondTo(msg),
				Error:   err,
			}, nil)
			return
		}
	default:
		for _, dir := range msg.Directories {
			if len(w.worker.Actions) > 0 {
				remaining = append(remaining, dir)
				continue
			}
			log.Tracef("Getting status of directory %s", dir)
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
			}
			w.selected = status
		}
		w.worker.PostMessage(&types.DirectoryInfo{
			Info: &models.DirectoryInfo{
				Flags:          status.Flags,
				Name:           status.Name,
				ReadOnly:       status.ReadOnly,
				AccurateCounts: true,

				Exists: int(status.Messages),
				Recent: int(status.Recent),
				Unseen: int(status.Unseen),
				Caps:   w.caps,
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
