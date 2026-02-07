package jmap

import (
	"context"
	"fmt"

	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
	"git.sr.ht/~rockorager/go-jmap"
	"git.sr.ht/~rockorager/go-jmap/mail/email"
	"git.sr.ht/~rockorager/go-jmap/mail/mailbox"
)

func (w *JMAPWorker) updateFlags(ctx context.Context, uids []models.UID, flags models.Flags, enable bool) error {
	var req jmap.Request
	patches := make(map[jmap.ID]jmap.Patch)

	for _, uid := range uids {
		patch := jmap.Patch{}
		for kw := range flagsToKeywords(flags) {
			path := fmt.Sprintf("keywords/%s", kw)
			if enable {
				patch[path] = true
			} else {
				patch[path] = nil
			}
		}
		patches[jmap.ID(uid)] = patch
	}

	req.Invoke(&email.Set{
		Account: w.AccountId(),
		Update:  patches,
	})

	resp, err := w.Do(ctx, &req)
	if err != nil {
		return err
	}

	err = checkNotUpdated(resp)
	if err != nil {
		return err
	}

	// If we didn't get an update error, all methods succeeded. We can
	// update the cache and UI now. We don't update the email state so that
	// we still grab an updated set from the update channel
	for _, uid := range uids {
		jid := jmap.ID(uid)
		m, err := w.cache.GetEmail(jid)
		if err != nil {
			// We'll get this from the update channel
			continue
		}
		if enable {
			for kw := range flagsToKeywords(flags) {
				m.Keywords[kw] = true
			}
		} else {
			for kw := range flagsToKeywords(flags) {
				delete(m.Keywords, kw)
			}
		}
		err = w.cache.PutEmail(jid, m)
		if err != nil {
			w.w.Warnf("PutEmail: %s", err)
		}
		// Get the UI updated immediately
		for mboxId := range m.MailboxIDs {
			dir, ok := w.mbox2dir[mboxId]
			if !ok {
				continue
			}
			info := w.translateMsgInfo(m, dir)
			w.w.PostMessage(&types.MessageInfo{
				Info: info,
			}, nil)
		}
	}

	return nil
}

func (w *JMAPWorker) moveCopy(ctx context.Context, uids []models.UID, srcDir, destDir string, deleteSrc bool) error {
	var req jmap.Request

	patches := make(map[jmap.ID]jmap.Patch)

	srcMbox, err := w.getMbox(srcDir)
	if err != nil {
		return err
	}
	destMbox, err := w.getMbox(destDir)
	if err != nil && destDir != "" {
		return fmt.Errorf("unknown destination mailbox")
	}
	if destMbox != nil && destMbox.ID == srcMbox.ID {
		return fmt.Errorf("cannot move to current mailbox")
	}

	patch := w.moveCopyPatch(srcMbox, destMbox, deleteSrc)
	for _, uid := range uids {
		patches[jmap.ID(uid)] = patch
	}

	req.Invoke(&email.Set{
		Account: w.AccountId(),
		Update:  patches,
	})

	resp, err := w.Do(ctx, &req)
	if err != nil {
		return err
	}

	return checkNotUpdated(resp)
}

func (w *JMAPWorker) moveCopyPatch(src, dest *mailbox.Mailbox, deleteSrc bool) jmap.Patch {
	patch := jmap.Patch{}

	if deleteSrc {
		patch[w.mboxPatch(src.ID)] = nil
	}
	if dest == nil && deleteSrc {
		patch[w.rolePatch(mailbox.RoleTrash)] = true
	} else if dest != nil {
		patch[w.mboxPatch(dest.ID)] = true
	}

	return patch
}

func (w *JMAPWorker) mboxPatch(mbox jmap.ID) string {
	return fmt.Sprintf("mailboxIds/%s", mbox)
}

func (w *JMAPWorker) rolePatch(role mailbox.Role) string {
	return fmt.Sprintf("mailboxIds/%s", w.roles[role])
}

func (w *JMAPWorker) handleModifyLabels(msg *types.ModifyLabels) error {
	var req jmap.Request
	patches := make(map[jmap.ID]jmap.Patch)

	for _, uid := range msg.Uids {
		email, err := w.cache.GetEmail(jmap.ID(uid))
		if err != nil {
			return fmt.Errorf("email not in cache: %w", err)
		}

		patch := jmap.Patch{}

		for _, a := range msg.Add {
			mboxId, ok := w.dir2mbox[a]
			if !ok {
				return fmt.Errorf("unknown label: %q", a)
			}
			patch[w.mboxPatch(mboxId)] = true
		}
		for _, r := range msg.Remove {
			mboxId, ok := w.dir2mbox[r]
			if !ok {
				return fmt.Errorf("unknown label: %q", r)
			}
			patch[w.mboxPatch(mboxId)] = nil
		}
		for _, t := range msg.Toggle {
			mboxId, ok := w.dir2mbox[t]
			if !ok {
				return fmt.Errorf("unknown label: %q", t)
			}
			if email.MailboxIDs[mboxId] {
				patch[w.mboxPatch(mboxId)] = nil
			} else {
				patch[w.mboxPatch(mboxId)] = true
			}
		}

		patches[jmap.ID(uid)] = patch
	}

	req.Invoke(&email.Set{
		Account: w.AccountId(),
		Update:  patches,
	})

	resp, err := w.Do(msg.Context(), &req)
	if err != nil {
		return err
	}

	return checkNotUpdated(resp)
}

func checkNotUpdated(resp *jmap.Response) error {
	for _, inv := range resp.Responses {
		switch r := inv.Args.(type) {
		case *email.SetResponse:
			for _, err := range r.NotUpdated {
				return wrapSetError(err)
			}
		case *jmap.MethodError:
			return wrapMethodError(r)
		}
	}
	return nil
}

func (w *JMAPWorker) handleAppendMessage(msg *types.AppendMessage) error {
	dest, ok := w.dir2mbox[msg.Destination]
	if !ok {
		return fmt.Errorf("unknown destination mailbox")
	}

	// Upload the message
	blob, err := w.Upload(msg.Reader)
	if err != nil {
		return err
	}

	var req jmap.Request

	// Import the blob into specified directory
	req.Invoke(&email.Import{
		Account: w.AccountId(),
		Emails: map[string]*email.EmailImport{
			"aerc": {
				BlobID:     blob.ID,
				MailboxIDs: map[jmap.ID]bool{dest: true},
				Keywords:   flagsToKeywords(msg.Flags),
			},
		},
	})

	resp, err := w.Do(msg.Context(), &req)
	if err != nil {
		return err
	}

	for _, inv := range resp.Responses {
		switch r := inv.Args.(type) {
		case *email.ImportResponse:
			if err, ok := r.NotCreated["aerc"]; ok {
				return wrapSetError(err)
			}
		case *jmap.MethodError:
			return wrapMethodError(r)
		}
	}

	return nil
}
