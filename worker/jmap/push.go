package jmap

import (
	"fmt"
	"sort"
	"time"

	"git.sr.ht/~rjarry/aerc/log"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/jmap/cache"
	"git.sr.ht/~rjarry/aerc/worker/types"
	"git.sr.ht/~rockorager/go-jmap"
	"git.sr.ht/~rockorager/go-jmap/core/push"
	"git.sr.ht/~rockorager/go-jmap/mail/email"
	"git.sr.ht/~rockorager/go-jmap/mail/mailbox"
)

func (w *JMAPWorker) monitorChanges() {
	events := push.EventSource{
		Client:  w.client,
		Handler: w.handleChange,
		Ping:    uint(w.config.serverPing.Seconds()),
	}

	w.stop = make(chan struct{})
	go func() {
		defer log.PanicHandler()
		<-w.stop
		w.w.Errorf("listen stopping")
		w.stop = nil
		events.Close()
	}()

	for w.stop != nil {
		w.w.Debugf("listening for changes")
		err := events.Listen()
		if err != nil {
			w.w.PostMessage(&types.Error{
				Error: fmt.Errorf("jmap listen: %w", err),
			}, nil)
			time.Sleep(5 * time.Second)
		}
	}
}

func (w *JMAPWorker) handleChange(s *jmap.StateChange) {
	changed, ok := s.Changed[w.accountId]
	if !ok {
		return
	}
	w.w.Debugf("state change %#v", changed)
	w.changes <- changed
}

func (w *JMAPWorker) refresh(newState jmap.TypeState) error {
	var req jmap.Request

	mboxState, err := w.cache.GetMailboxState()
	if err != nil {
		w.w.Debugf("GetMailboxState: %s", err)
	}
	if mboxState != "" && newState["Mailbox"] != mboxState {
		callID := req.Invoke(&mailbox.Changes{
			Account:    w.accountId,
			SinceState: mboxState,
		})
		req.Invoke(&mailbox.Get{
			Account: w.accountId,
			ReferenceIDs: &jmap.ResultReference{
				ResultOf: callID,
				Name:     "Mailbox/changes",
				Path:     "/created",
			},
		})
		req.Invoke(&mailbox.Get{
			Account: w.accountId,
			ReferenceIDs: &jmap.ResultReference{
				ResultOf: callID,
				Name:     "Mailbox/changes",
				Path:     "/updated",
			},
		})
	}

	emailState, err := w.cache.GetEmailState()
	if err != nil {
		w.w.Debugf("GetEmailState: %s", err)
	}
	queryChangesCalls := make(map[string]jmap.ID)
	folderContents := make(map[jmap.ID]*cache.FolderContents)
	ids, _ := w.cache.GetMailboxList()
	mboxes := make(map[jmap.ID]*mailbox.Mailbox)
	for _, id := range ids {
		mbox, err := w.cache.GetMailbox(id)
		if err != nil {
			w.w.Warnf("GetMailbox: %s", err)
			continue
		}
		if mbox.Role == mailbox.RoleArchive && w.config.useLabels {
			mboxes[""] = &mailbox.Mailbox{
				Name: w.config.allMail,
				Role: mailbox.RoleAll,
			}
		} else {
			mboxes[id] = mbox
		}
	}
	if emailState != "" && newState["Email"] != emailState {
		callID := req.Invoke(&email.Changes{
			Account:    w.accountId,
			SinceState: emailState,
		})
		req.Invoke(&email.Get{
			Account:    w.accountId,
			Properties: headersProperties,
			ReferenceIDs: &jmap.ResultReference{
				ResultOf: callID,
				Name:     "Email/changes",
				Path:     "/updated",
			},
		})

		for id := range mboxes {
			contents, err := w.cache.GetFolderContents(id)
			if err != nil {
				continue
			}
			callID = req.Invoke(&email.QueryChanges{
				Account:         w.accountId,
				Filter:          contents.Filter,
				Sort:            contents.Sort,
				SinceQueryState: contents.QueryState,
			})
			queryChangesCalls[callID] = id
			folderContents[id] = contents
		}
	}

	if len(req.Calls) == 0 {
		return nil
	}

	resp, err := w.Do(&req)
	if err != nil {
		return err
	}

	var changedMboxIds []jmap.ID
	var labelsChanged bool

	for _, inv := range resp.Responses {
		switch r := inv.Args.(type) {
		case *mailbox.ChangesResponse:
			for _, id := range r.Destroyed {
				dir, ok := w.mbox2dir[id]
				if ok {
					w.w.PostMessage(&types.RemoveDirectory{
						Directory: dir,
					}, nil)
				}
				w.deleteMbox(id)
				err = w.cache.DeleteMailbox(id)
				if err != nil {
					w.w.Warnf("DeleteMailbox: %s", err)
				}
				labelsChanged = true
			}
			err = w.cache.PutMailboxState(r.NewState)
			if err != nil {
				w.w.Warnf("PutMailboxState: %s", err)
			}

		case *mailbox.GetResponse:
			for _, mbox := range r.List {
				changedMboxIds = append(changedMboxIds, mbox.ID)
				mboxes[mbox.ID] = mbox
				err = w.cache.PutMailbox(mbox.ID, mbox)
				if err != nil {
					w.w.Warnf("PutMailbox: %s", err)
				}
			}
			err = w.cache.PutMailboxState(r.State)
			if err != nil {
				w.w.Warnf("PutMailboxState: %s", err)
			}

		case *email.QueryChangesResponse:
			mboxId := queryChangesCalls[inv.CallID]
			contents := folderContents[mboxId]

			removed := make(map[jmap.ID]bool)
			for _, id := range r.Removed {
				removed[id] = true
			}
			added := make(map[int]jmap.ID)
			for _, add := range r.Added {
				added[int(add.Index)] = add.ID
			}
			w.w.Debugf("%q: %d added, %d removed",
				w.mbox2dir[mboxId], len(added), len(removed))
			n := len(contents.MessageIDs) - len(removed) + len(added)
			if n < 0 {
				w.w.Errorf("bug: invalid folder contents state")
				err = w.cache.DeleteFolderContents(mboxId)
				if err != nil {
					w.w.Warnf("DeleteFolderContents: %s", err)
				}
				continue
			}
			ids = make([]jmap.ID, 0, n)
			i := 0
			for _, id := range contents.MessageIDs {
				if removed[id] {
					continue
				}
				if addedId, ok := added[i]; ok {
					ids = append(ids, addedId)
					delete(added, i)
					i += 1
				}
				ids = append(ids, id)
				i += 1
			}
			for _, id := range added {
				ids = append(ids, id)
			}
			contents.MessageIDs = ids
			contents.QueryState = r.NewQueryState

			err = w.cache.PutFolderContents(mboxId, contents)
			if err != nil {
				w.w.Warnf("PutFolderContents: %s", err)
			}

			if w.selectedMbox == mboxId {
				uids := make([]uint32, 0, len(ids))
				for _, id := range ids {
					uid := w.uidStore.GetOrInsert(string(id))
					uids = append(uids, uid)
				}
				w.w.PostMessage(&types.DirectoryContents{
					Uids: uids,
				}, nil)
			}

		case *email.GetResponse:
			selectedIds := make(map[jmap.ID]bool)
			contents, ok := folderContents[w.selectedMbox]
			if ok {
				for _, id := range contents.MessageIDs {
					selectedIds[id] = true
				}
			}
			for _, m := range r.List {
				err = w.cache.PutEmail(m.ID, m)
				if err != nil {
					w.w.Warnf("PutEmail: %s", err)
				}
				if selectedIds[m.ID] {
					w.w.PostMessage(&types.MessageInfo{
						Info: w.translateMsgInfo(m),
					}, nil)
				}
			}
			err = w.cache.PutEmailState(r.State)
			if err != nil {
				w.w.Warnf("PutEmailState: %s", err)
			}

		case *jmap.MethodError:
			w.w.Errorf("%s: %s", wrapMethodError(r))
			if inv.Name == "Email/queryChanges" {
				id := queryChangesCalls[inv.CallID]
				w.w.Infof("flushing %q contents from cache",
					w.mbox2dir[id])
				err := w.cache.DeleteFolderContents(id)
				if err != nil {
					w.w.Warnf("DeleteFolderContents: %s", err)
				}
			}
		}
	}

	for _, id := range changedMboxIds {
		mbox := mboxes[id]
		if mbox.Role == mailbox.RoleArchive && w.config.useLabels {
			continue
		}
		newDir := w.MailboxPath(mbox)
		dir, ok := w.mbox2dir[id]
		if ok {
			// updated
			if newDir == dir {
				w.deleteMbox(id)
				w.addMbox(mbox, dir)
				w.w.PostMessage(&types.DirectoryInfo{
					Info: &models.DirectoryInfo{
						Name:   dir,
						Exists: int(mbox.TotalEmails),
						Unseen: int(mbox.UnreadEmails),
					},
				}, nil)
				continue
			} else {
				// renamed mailbox
				w.deleteMbox(id)
				w.w.PostMessage(&types.RemoveDirectory{
					Directory: dir,
				}, nil)
				dir = newDir
			}
		}
		// new mailbox
		w.addMbox(mbox, dir)
		w.w.PostMessage(&types.Directory{
			Dir: &models.Directory{
				Name:   dir,
				Exists: int(mbox.TotalEmails),
				Unseen: int(mbox.UnreadEmails),
				Role:   jmapRole2aerc[mbox.Role],
			},
		}, nil)
		labelsChanged = true
	}

	if w.config.useLabels && labelsChanged {
		labels := make([]string, 0, len(w.dir2mbox))
		for dir := range w.dir2mbox {
			labels = append(labels, dir)
		}
		sort.Strings(labels)
		w.w.PostMessage(&types.LabelList{Labels: labels}, nil)
	}

	return nil
}
