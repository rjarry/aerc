package jmap

import (
	"errors"
	"fmt"
	"path"
	"sort"

	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/jmap/cache"
	"git.sr.ht/~rjarry/aerc/worker/types"
	"git.sr.ht/~rockorager/go-jmap"
	"git.sr.ht/~rockorager/go-jmap/mail/email"
	"git.sr.ht/~rockorager/go-jmap/mail/mailbox"
)

func (w *JMAPWorker) handleListDirectories(msg *types.ListDirectories) error {
	var ids, missing []jmap.ID
	var labels []string
	var mboxes map[jmap.ID]*mailbox.Mailbox

	mboxes = make(map[jmap.ID]*mailbox.Mailbox)

	// If we can't get the cached mailbox state, at worst, we will just
	// query information we might already know
	cachedMailboxState, err := w.cache.GetMailboxState()
	if err != nil {
		w.w.Warnf("GetMailboxState: %s", err)
	}

	mboxIds, err := w.cache.GetMailboxList()
	if err == nil {
		for _, id := range mboxIds {
			mbox, err := w.cache.GetMailbox(id)
			if err != nil {
				w.w.Warnf("GetMailbox: %s", err)
				missing = append(missing, id)
				continue
			}
			mboxes[id] = mbox
			ids = append(ids, id)
		}
	}

	if cachedMailboxState == "" || len(missing) > 0 {
		var req jmap.Request

		req.Invoke(&mailbox.Get{Account: w.AccountId()})
		resp, err := w.Do(msg.Context(), &req)
		if err != nil {
			return err
		}

		mboxes = make(map[jmap.ID]*mailbox.Mailbox)
		ids = make([]jmap.ID, 0)

		for _, inv := range resp.Responses {
			switch r := inv.Args.(type) {
			case *mailbox.GetResponse:
				for _, mbox := range r.List {
					mboxes[mbox.ID] = mbox
					ids = append(ids, mbox.ID)
					err = w.cache.PutMailbox(mbox.ID, mbox)
					if err != nil {
						w.w.Warnf("PutMailbox: %s", err)
					}
				}
				err = w.cache.PutMailboxList(ids)
				if err != nil {
					w.w.Warnf("PutMailboxList: %s", err)
				}
				err = w.cache.PutMailboxState(r.State)
				if err != nil {
					w.w.Warnf("PutMailboxState: %s", err)
				}
			case *jmap.MethodError:
				return wrapMethodError(r)
			}
		}
	}

	if len(mboxes) == 0 {
		return errors.New("no mailboxes")
	}

	w.mboxes = mboxes

	for _, mbox := range mboxes {
		w.addMbox(mbox)
		labels = append(labels, w.mbox2dir[mbox.ID])
	}
	if w.config.useLabels {
		sort.Strings(labels)
		w.w.PostMessage(&types.LabelList{Labels: labels}, nil)
	}

	for _, id := range ids {
		mbox := mboxes[id]
		if mbox.Role == mailbox.RoleArchive && w.config.useLabels {
			// replace archive with virtual all-mail folder
			mbox = &mailbox.Mailbox{
				Name: w.config.allMail,
				Role: mailbox.RoleAll,
			}
			w.addMbox(mbox)
		}
		w.w.PostMessage(&types.Directory{
			Message: types.RespondTo(msg),
			Dir: &models.Directory{
				Name:   w.mbox2dir[mbox.ID],
				Exists: int(mbox.TotalEmails),
				Unseen: int(mbox.UnreadEmails),
				Role:   jmapRole2aerc[mbox.Role],
			},
		}, nil)
	}

	return nil
}

func (w *JMAPWorker) handleOpenDirectory(msg *types.OpenDirectory) error {
	id, ok := w.dir2mbox[msg.Directory]
	if !ok {
		return fmt.Errorf("unknown directory: %s", msg.Directory)
	}
	w.selectedMbox = id
	return nil
}

func (w *JMAPWorker) handleFetchDirectoryContents(msg *types.FetchDirectoryContents) error {
	contents, err := w.cache.GetFolderContents(w.selectedMbox)
	if err != nil {
		contents = &cache.FolderContents{
			MailboxID: w.selectedMbox,
		}
	}

	if contents.NeedsRefresh(msg.Filter, msg.SortCriteria) {
		var req jmap.Request

		req.Invoke(&email.Query{
			Account: w.AccountId(),
			Filter:  w.translateSearch(w.selectedMbox, msg.Filter),
			Sort:    translateSort(msg.SortCriteria),
		})
		resp, err := w.Do(msg.Context(), &req)
		if err != nil {
			return err
		}
		var canCalculateChanges bool
		for _, inv := range resp.Responses {
			switch r := inv.Args.(type) {
			case *email.QueryResponse:
				contents.Sort = msg.SortCriteria
				contents.Filter = msg.Filter
				contents.QueryState = r.QueryState
				contents.MessageIDs = r.IDs
				canCalculateChanges = r.CanCalculateChanges
			case *jmap.MethodError:
				return wrapMethodError(r)
			}
		}
		if canCalculateChanges {
			err = w.cache.PutFolderContents(w.selectedMbox, contents)
			if err != nil {
				w.w.Warnf("PutFolderContents: %s", err)
			}
		} else {
			w.w.Debugf("%q: server cannot calculate changes, flushing cache",
				w.mbox2dir[w.selectedMbox])
			err = w.cache.DeleteFolderContents(w.selectedMbox)
			if err != nil {
				w.w.Warnf("DeleteFolderContents: %s", err)
			}
		}
	}

	uids := make([]models.UID, 0, len(contents.MessageIDs))
	for _, id := range contents.MessageIDs {
		uids = append(uids, models.UID(id))
	}
	w.w.PostMessage(&types.DirectoryContents{
		Message: types.RespondTo(msg),
		Uids:    uids,
	}, nil)

	return nil
}

func (w *JMAPWorker) handleSearchDirectory(msg *types.SearchDirectory) error {
	var req jmap.Request

	req.Invoke(&email.Query{
		Account: w.AccountId(),
		Filter:  w.translateSearch(w.selectedMbox, msg.Criteria),
	})

	resp, err := w.Do(msg.Context(), &req)
	if err != nil {
		return err
	}

	for _, inv := range resp.Responses {
		switch r := inv.Args.(type) {
		case *email.QueryResponse:
			var uids []models.UID
			for _, id := range r.IDs {
				uids = append(uids, models.UID(id))
			}
			w.w.PostMessage(&types.SearchResults{
				Message: types.RespondTo(msg),
				Uids:    uids,
			}, nil)
		case *jmap.MethodError:
			return wrapMethodError(r)
		}
	}

	return nil
}

func (w *JMAPWorker) handleCreateDirectory(msg *types.CreateDirectory) error {
	var req jmap.Request
	var parentId, id jmap.ID

	if id, ok := w.dir2mbox[msg.Directory]; ok {
		// directory already exists
		mbox, err := w.cache.GetMailbox(id)
		if err != nil {
			return err
		}
		if mbox.Role == mailbox.RoleArchive && w.config.useLabels {
			return types.ErrNoop
		}
		return nil
	}
	if parent := path.Dir(msg.Directory); parent != "" && parent != "." {
		var ok bool
		if parentId, ok = w.dir2mbox[parent]; !ok {
			return fmt.Errorf(
				"parent mailbox %q does not exist", parent)
		}
	}
	name := path.Base(msg.Directory)
	id = jmap.ID(msg.Directory)

	req.Invoke(&mailbox.Set{
		Account: w.AccountId(),
		Create: map[jmap.ID]*mailbox.Mailbox{
			id: {
				ParentID: parentId,
				Name:     name,
			},
		},
	})

	resp, err := w.Do(msg.Context(), &req)
	if err != nil {
		return err
	}
	for _, inv := range resp.Responses {
		switch r := inv.Args.(type) {
		case *mailbox.SetResponse:
			if err := r.NotCreated[id]; err != nil {
				e := wrapSetError(err)
				if msg.Quiet {
					w.w.Warnf("mailbox creation failed: %s", e)
				} else {
					return e
				}
			}
		case *jmap.MethodError:
			return wrapMethodError(r)
		}
	}

	return nil
}

func (w *JMAPWorker) handleRemoveDirectory(msg *types.RemoveDirectory) error {
	var req jmap.Request

	id, ok := w.dir2mbox[msg.Directory]
	if !ok {
		return fmt.Errorf("unknown mailbox: %s", msg.Directory)
	}

	req.Invoke(&mailbox.Set{
		Account:               w.AccountId(),
		Destroy:               []jmap.ID{id},
		OnDestroyRemoveEmails: msg.Quiet,
	})

	resp, err := w.Do(msg.Context(), &req)
	if err != nil {
		return err
	}
	for _, inv := range resp.Responses {
		switch r := inv.Args.(type) {
		case *mailbox.SetResponse:
			if err := r.NotDestroyed[id]; err != nil {
				return wrapSetError(err)
			}
		case *jmap.MethodError:
			return wrapMethodError(r)
		}
	}

	return nil
}

func translateSort(criteria []*types.SortCriterion) []*email.SortComparator {
	sort := make([]*email.SortComparator, 0, len(criteria))
	if len(criteria) == 0 {
		criteria = []*types.SortCriterion{
			{Field: types.SortArrival, Reverse: true},
		}
	}
	for _, s := range criteria {
		var cmp email.SortComparator
		switch s.Field {
		case types.SortArrival:
			cmp.Property = "receivedAt"
		case types.SortCc:
			cmp.Property = "cc"
		case types.SortDate:
			cmp.Property = "receivedAt"
		case types.SortFrom:
			cmp.Property = "from"
		case types.SortRead:
			cmp.Keyword = "$seen"
		case types.SortSize:
			cmp.Property = "size"
		case types.SortSubject:
			cmp.Property = "subject"
		case types.SortTo:
			cmp.Property = "to"
		default:
			continue
		}
		cmp.IsAscending = s.Reverse
		sort = append(sort, &cmp)
	}

	return sort
}
