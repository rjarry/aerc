package jmap

import (
	"context"
	"fmt"
	"sort"
	"time"

	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/jmap/cache"
	"git.sr.ht/~rjarry/aerc/worker/types"
	"git.sr.ht/~rockorager/go-jmap"
	"git.sr.ht/~rockorager/go-jmap/core/push"
	"git.sr.ht/~rockorager/go-jmap/mail/email"
	"git.sr.ht/~rockorager/go-jmap/mail/mailbox"
	"git.sr.ht/~rockorager/go-jmap/mail/thread"
)

func (w *JMAPWorker) monitorChanges() {
	defer log.PanicHandler()

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
	changed, ok := s.Changed[w.AccountId()]
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
			Account:    w.AccountId(),
			SinceState: mboxState,
		})
		req.Invoke(&mailbox.Get{
			Account: w.AccountId(),
			ReferenceIDs: &jmap.ResultReference{
				ResultOf: callID,
				Name:     "Mailbox/changes",
				Path:     "/created",
			},
		})
		req.Invoke(&mailbox.Get{
			Account: w.AccountId(),
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
	emailUpdated := ""
	emailCreated := ""
	if emailState != "" && newState["Email"] != emailState {
		callID := req.Invoke(&email.Changes{
			Account:    w.AccountId(),
			SinceState: emailState,
		})
		emailUpdated = req.Invoke(&email.Get{
			Account:        w.AccountId(),
			Properties:     emailProperties,
			BodyProperties: bodyProperties,
			ReferenceIDs: &jmap.ResultReference{
				ResultOf: callID,
				Name:     "Email/changes",
				Path:     "/updated",
			},
		})

		emailCreated = req.Invoke(&email.Get{
			Account:        w.AccountId(),
			Properties:     emailProperties,
			BodyProperties: bodyProperties,
			ReferenceIDs: &jmap.ResultReference{
				ResultOf: callID,
				Name:     "Email/changes",
				Path:     "/created",
			},
		})
	}

	threadState, err := w.cache.GetThreadState()
	if err != nil {
		w.w.Debugf("GetThreadState: %s", err)
	}
	if threadState != "" && newState["Thread"] != threadState {
		callID := req.Invoke(&thread.Changes{
			Account:    w.AccountId(),
			SinceState: threadState,
		})
		req.Invoke(&thread.Get{
			Account: w.AccountId(),
			ReferenceIDs: &jmap.ResultReference{
				ResultOf: callID,
				Name:     "Thread/changes",
				Path:     "/created",
			},
		})
		req.Invoke(&thread.Get{
			Account: w.AccountId(),
			ReferenceIDs: &jmap.ResultReference{
				ResultOf: callID,
				Name:     "Thread/changes",
				Path:     "/updated",
			},
		})
	}

	if len(req.Calls) == 0 {
		return nil
	}

	resp, err := w.Do(context.TODO(), &req)
	if err != nil {
		return err
	}

	var changedMboxIds []jmap.ID
	var labelsChanged bool
	// threadEmails are email IDs from threads which changed or were
	// created
	var threadEmails []jmap.ID

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

		case *thread.ChangesResponse:
			for _, id := range r.Destroyed {
				err = w.cache.DeleteThread(id)
				if err != nil {
					w.w.Warnf("DeleteThread: %s", err)
				}
			}
			err = w.cache.PutThreadState(r.NewState)
			if err != nil {
				w.w.Warnf("PutThreadState: %s", err)
			}

		case *thread.GetResponse:
			for _, thread := range r.List {
				err = w.cache.PutThread(thread.ID, thread.EmailIDs)
				if err != nil {
					w.w.Warnf("PutThread: %s", err)
				}
				// We keep the list of all emails and check in a
				// subsequent request which ones we need to
				// fetch
				threadEmails = append(threadEmails, thread.EmailIDs...)
			}
			err = w.cache.PutThreadState(r.State)
			if err != nil {
				w.w.Warnf("PutThreadState: %s", err)
			}

		case *email.GetResponse:
			switch inv.CallID {
			case emailUpdated:
				for _, m := range r.List {
					err = w.cache.PutEmail(m.ID, m)
					if err != nil {
						w.w.Warnf("PutEmail: %s", err)
					}
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
				err = w.cache.PutEmailState(r.State)
				if err != nil {
					w.w.Warnf("PutEmailState: %s", err)
				}
			case emailCreated:
				for _, m := range r.List {
					err = w.cache.PutEmail(m.ID, m)
					if err != nil {
						w.w.Warnf("PutEmail: %s", err)
					}
					for mboxId := range m.MailboxIDs {
						dir, ok := w.mbox2dir[mboxId]
						if !ok {
							continue
						}
						info := w.translateMsgInfo(m, dir)
						// Set recent on created messages so we
						// get a notification
						info.Flags |= models.RecentFlag
						w.w.PostMessage(&types.MessageInfo{
							Info: info,
						}, nil)
					}
				}
				err = w.cache.PutEmailState(r.State)
				if err != nil {
					w.w.Warnf("PutEmailState: %s", err)
				}
			}

		case *jmap.MethodError:
			w.w.Errorf("%s: %s", wrapMethodError(r))
		}
	}

	var updatedMboxes []jmap.ID
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
				w.addMbox(mbox)
				w.w.PostMessage(&types.DirectoryInfo{
					Info: &models.DirectoryInfo{
						Name:   dir,
						Exists: int(mbox.TotalEmails),
						Unseen: int(mbox.UnreadEmails),
					},
				}, nil)

				updatedMboxes = append(updatedMboxes, id)
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
		w.addMbox(mbox)
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

	return w.refreshQueriesAndThreads(updatedMboxes, threadEmails)
}

// refreshQueriesAndThreads updates the cached query for any mailbox which was updated
func (w *JMAPWorker) refreshQueriesAndThreads(
	updatedMboxes []jmap.ID,
	threadEmails []jmap.ID,
) error {
	if len(updatedMboxes) == 0 && len(threadEmails) == 0 {
		return nil
	}

	var req jmap.Request
	queryChangesCalls := make(map[string]jmap.ID)
	folderContents := make(map[jmap.ID]*cache.FolderContents)

	for _, id := range updatedMboxes {
		contents, err := w.cache.GetFolderContents(id)
		if err != nil {
			continue
		}
		callID := req.Invoke(&email.QueryChanges{
			Account:         w.AccountId(),
			Filter:          w.translateSearch(id, contents.Filter),
			Sort:            translateSort(contents.Sort),
			SinceQueryState: contents.QueryState,
		})
		queryChangesCalls[callID] = id
		folderContents[id] = contents
	}

	emailsToFetch := []jmap.ID{}
	for _, id := range threadEmails {
		if w.cache.HasEmail(id) {
			continue
		}
		emailsToFetch = append(emailsToFetch, id)
	}

	if len(emailsToFetch) != 0 {
		req.Invoke(&email.Get{
			Account:        w.AccountId(),
			Properties:     emailProperties,
			BodyProperties: bodyProperties,
			IDs:            emailsToFetch,
		})
	}

	resp, err := w.Do(context.TODO(), &req)
	if err != nil {
		return err
	}

	for _, inv := range resp.Responses {
		switch r := inv.Args.(type) {
		case *email.QueryChangesResponse:
			mboxId := queryChangesCalls[inv.CallID]
			contents := folderContents[mboxId]
			dir := w.mbox2dir[mboxId]

			removed := make(map[jmap.ID]bool)
			for _, id := range r.Removed {
				removed[id] = true
			}
			added := make(map[int]jmap.ID)
			for _, add := range r.Added {
				added[int(add.Index)] = add.ID
			}
			w.w.Debugf("%q: %d added, %d removed", dir, len(added), len(removed))
			n := len(contents.MessageIDs) - len(removed) + len(added)
			if n < 0 {
				w.w.Errorf("bug: invalid folder contents state")
				err = w.cache.DeleteFolderContents(mboxId)
				if err != nil {
					w.w.Warnf("DeleteFolderContents: %s", err)
				}
				continue
			}
			ids := make([]jmap.ID, 0, n)
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

			// Post MessageInfo with Index for each added message
			for _, add := range r.Added {
				m, err := w.cache.GetEmail(add.ID)
				if err != nil {
					continue
				}
				info := w.translateMsgInfo(m, dir)
				info.Flags |= models.RecentFlag
				idx := int(add.Index)
				info.Index = &idx
				w.w.PostMessage(&types.MessageInfo{
					Info: info,
				}, nil)
			}

			// Post MessagesDeleted for removed messages
			if len(r.Removed) > 0 {
				deletedUids := make([]models.UID, 0, len(r.Removed))
				for _, id := range r.Removed {
					deletedUids = append(deletedUids, models.UID(id))
				}
				w.w.PostMessage(&types.MessagesDeleted{
					Directory: dir,
					Uids:      deletedUids,
				}, nil)
			}

		case *email.GetResponse:
			for _, m := range r.List {
				err = w.cache.PutEmail(m.ID, m)
				if err != nil {
					w.w.Warnf("PutEmail: %s", err)
				}
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
	return nil
}
