package jmap

import (
	"fmt"

	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
	"git.sr.ht/~rockorager/go-jmap"
	"git.sr.ht/~rockorager/go-jmap/mail/email"
	"git.sr.ht/~rockorager/go-jmap/mail/mailbox"
)

func (w *JMAPWorker) updateFlags(uids []uint32, flags models.Flags, enable bool) error {
	var req jmap.Request
	patches := make(map[jmap.ID]jmap.Patch)

	for _, uid := range uids {
		id, ok := w.uidStore.GetKey(uid)
		if !ok {
			return fmt.Errorf("bug: unknown uid %d", uid)
		}
		patch := jmap.Patch{}
		for kw := range flagsToKeywords(flags) {
			path := fmt.Sprintf("keywords/%s", kw)
			if enable {
				patch[path] = true
			} else {
				patch[path] = nil
			}
		}
		patches[jmap.ID(id)] = patch
	}

	req.Invoke(&email.Set{
		Account: w.accountId,
		Update:  patches,
	})

	resp, err := w.Do(&req)
	if err != nil {
		return err
	}

	return checkNotUpdated(resp)
}

func (w *JMAPWorker) moveCopy(uids []uint32, destDir string, deleteSrc bool) error {
	var req jmap.Request
	var destMbox jmap.ID
	var destroy []jmap.ID
	var ok bool

	patches := make(map[jmap.ID]jmap.Patch)

	destMbox, ok = w.dir2mbox[destDir]
	if !ok && destDir != "" {
		return fmt.Errorf("unknown destination mailbox")
	}
	if destMbox != "" && destMbox == w.selectedMbox {
		return fmt.Errorf("cannot move to current mailbox")
	}

	for _, uid := range uids {
		dest := destMbox
		id, ok := w.uidStore.GetKey(uid)
		if !ok {
			return fmt.Errorf("bug: unknown uid %d", uid)
		}
		mail, err := w.cache.GetEmail(jmap.ID(id))
		if err != nil {
			return fmt.Errorf("bug: unknown message id %s: %w", id, err)
		}

		patch := w.moveCopyPatch(mail, dest, deleteSrc)
		if len(patch) == 0 {
			destroy = append(destroy, mail.ID)
			w.w.Debugf("destroying <%s>", mail.MessageID[0])
		} else {
			patches[jmap.ID(id)] = patch
		}
	}

	req.Invoke(&email.Set{
		Account: w.accountId,
		Update:  patches,
		Destroy: destroy,
	})

	resp, err := w.Do(&req)
	if err != nil {
		return err
	}

	return checkNotUpdated(resp)
}

func (w *JMAPWorker) moveCopyPatch(
	mail *email.Email, dest jmap.ID, deleteSrc bool,
) jmap.Patch {
	patch := jmap.Patch{}

	if dest == "" && deleteSrc && len(mail.MailboxIDs) == 1 {
		dest = w.roles[mailbox.RoleTrash]
	}
	if dest != "" && dest != w.selectedMbox {
		d := w.mbox2dir[dest]
		if deleteSrc {
			w.w.Debugf("moving <%s> to %q", mail.MessageID[0], d)
		} else {
			w.w.Debugf("copying <%s> to %q", mail.MessageID[0], d)
		}
		patch[w.mboxPatch(dest)] = true
	}
	if deleteSrc && len(patch) > 0 {
		switch {
		case w.selectedMbox != "":
			patch[w.mboxPatch(w.selectedMbox)] = nil
		case len(mail.MailboxIDs) == 1:
			// In "all mail" virtual mailbox and email is in
			// a single mailbox, "Move" it to the specified
			// destination
			patch = jmap.Patch{"mailboxIds": []jmap.ID{dest}}
		default:
			// In "all mail" virtual mailbox and email is in
			// multiple mailboxes. Since we cannot know what mailbox
			// to remove, try at least to remove role=inbox.
			patch[w.rolePatch(mailbox.RoleInbox)] = nil
		}
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
	patch := jmap.Patch{}

	for _, a := range msg.Add {
		mboxId, ok := w.dir2mbox[a]
		if !ok {
			return fmt.Errorf("unkown label: %q", a)
		}
		patch[w.mboxPatch(mboxId)] = true
	}
	for _, r := range msg.Remove {
		mboxId, ok := w.dir2mbox[r]
		if !ok {
			return fmt.Errorf("unkown label: %q", r)
		}
		patch[w.mboxPatch(mboxId)] = nil
	}

	patches := make(map[jmap.ID]jmap.Patch)

	for _, uid := range msg.Uids {
		id, ok := w.uidStore.GetKey(uid)
		if !ok {
			return fmt.Errorf("bug: unknown uid %d", uid)
		}
		patches[jmap.ID(id)] = patch
	}

	req.Invoke(&email.Set{
		Account: w.accountId,
		Update:  patches,
	})

	resp, err := w.Do(&req)
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
		Account: w.accountId,
		Emails: map[string]*email.EmailImport{
			"aerc": {
				BlobID:     blob.ID,
				MailboxIDs: map[jmap.ID]bool{dest: true},
				Keywords:   flagsToKeywords(msg.Flags),
			},
		},
	})

	resp, err := w.Do(&req)
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
