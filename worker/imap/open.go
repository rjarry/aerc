package imap

import (
	"sort"

	sortthread "github.com/emersion/go-imap-sortthread"

	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

func (imapw *IMAPWorker) handleOpenDirectory(msg *types.OpenDirectory) {
	imapw.worker.Debugf("Opening %s", msg.Directory)

	sel, err := imapw.client.Select(msg.Directory, false)
	if err != nil {
		imapw.worker.PostMessage(&types.Error{
			Message: types.RespondTo(msg),
			Error:   err,
		}, nil)
		return
	}
	select {
	case <-msg.Context.Done():
		imapw.worker.PostMessage(&types.Cancelled{Message: types.RespondTo(msg)}, nil)
	default:
		imapw.selected = sel
		imapw.worker.PostMessage(&types.Done{Message: types.RespondTo(msg)}, nil)
	}
}

func (imapw *IMAPWorker) handleFetchDirectoryContents(
	msg *types.FetchDirectoryContents,
) {
	if msg.Context.Err() != nil {
		imapw.worker.PostMessage(&types.Cancelled{
			Message: types.RespondTo(msg),
		}, nil)
		return
	}
	imapw.worker.Tracef("Fetching UID list")

	searchCriteria := translateSearch(msg.Filter)
	sortCriteria := translateSortCriterions(msg.SortCriteria)
	hasSortCriteria := len(sortCriteria) > 0

	var err error
	var uids []uint32

	// If the server supports the SORT extension, do the sorting server side
	switch {
	case imapw.caps.Sort && hasSortCriteria:
		uids, err = imapw.client.sort.UidSort(sortCriteria, searchCriteria)
		if err != nil {
			imapw.worker.PostMessage(&types.Error{
				Message: types.RespondTo(msg),
				Error:   err,
			}, nil)
			return
		}
		// copy in reverse as msgList displays backwards
		for i, j := 0, len(uids)-1; i < j; i, j = i+1, j-1 {
			uids[i], uids[j] = uids[j], uids[i]
		}
	default:
		if hasSortCriteria {
			imapw.worker.Warnf("SORT is not supported but requested: list messages by UID")
		}
		uids, err = imapw.client.UidSearch(searchCriteria)
		if err != nil {
			imapw.worker.PostMessage(&types.Error{
				Message: types.RespondTo(msg),
				Error:   err,
			}, nil)
			return
		}
	}
	if msg.Context.Err() != nil {
		imapw.worker.PostMessage(&types.Cancelled{
			Message: types.RespondTo(msg),
		}, nil)
		return
	}
	imapw.worker.Tracef("Found %d UIDs", len(uids))
	if msg.Filter == nil {
		// Only initialize if we are not filtering
		imapw.seqMap.Initialize(uids)
	}

	imapw.worker.PostMessage(&types.DirectoryContents{
		Message: types.RespondTo(msg),
		Uids:    models.Uint32ToUidList(uids),
	}, nil)
	imapw.worker.PostMessage(&types.Done{Message: types.RespondTo(msg)}, nil)
}

type sortFieldMapT map[types.SortField]sortthread.SortField

// caution, incomplete mapping
var sortFieldMap sortFieldMapT = sortFieldMapT{
	types.SortArrival: sortthread.SortArrival,
	types.SortCc:      sortthread.SortCc,
	types.SortDate:    sortthread.SortDate,
	types.SortFrom:    sortthread.SortFrom,
	types.SortSize:    sortthread.SortSize,
	types.SortSubject: sortthread.SortSubject,
	types.SortTo:      sortthread.SortTo,
}

func translateSortCriterions(
	cs []*types.SortCriterion,
) []sortthread.SortCriterion {
	result := make([]sortthread.SortCriterion, 0, len(cs))
	for _, c := range cs {
		if f, ok := sortFieldMap[c.Field]; ok {
			result = append(result, sortthread.SortCriterion{Field: f, Reverse: c.Reverse})
		}
	}
	return result
}

func (imapw *IMAPWorker) handleDirectoryThreaded(
	msg *types.FetchDirectoryThreaded,
) {
	if msg.Context.Err() != nil {
		imapw.worker.PostMessage(&types.Cancelled{
			Message: types.RespondTo(msg),
		}, nil)
		return
	}
	imapw.worker.Tracef("Fetching threaded UID list")

	searchCriteria := translateSearch(msg.Filter)
	threads, err := imapw.client.thread.UidThread(imapw.threadAlgorithm,
		searchCriteria)
	if err != nil {
		imapw.worker.PostMessage(&types.Error{
			Message: types.RespondTo(msg),
			Error:   err,
		}, nil)
		return
	}
	aercThreads, count := convertThreads(threads, nil)
	sort.Sort(types.ByUID(aercThreads))
	imapw.worker.Tracef("Found %d threaded messages", count)
	if msg.Filter == nil {
		// Only initialize if we are not filtering
		var uids []uint32
		for i := len(aercThreads) - 1; i >= 0; i-- {
			aercThreads[i].Walk(func(t *types.Thread, level int, currentErr error) error { //nolint:errcheck // error indicates skipped threads
				uids = append(uids, models.UidToUint32(t.Uid))
				return nil
			})
		}
		imapw.seqMap.Initialize(uids)
	}
	if msg.Context.Err() != nil {
		imapw.worker.PostMessage(&types.Cancelled{
			Message: types.RespondTo(msg),
		}, nil)
		return
	}
	imapw.worker.PostMessage(&types.DirectoryThreaded{
		Message: types.RespondTo(msg),
		Threads: aercThreads,
	}, nil)
	imapw.worker.PostMessage(&types.Done{Message: types.RespondTo(msg)}, nil)
}

func convertThreads(threads []*sortthread.Thread, parent *types.Thread) ([]*types.Thread, int) {
	if threads == nil {
		return nil, 0
	}
	conv := make([]*types.Thread, len(threads))
	count := 0

	for i := 0; i < len(threads); i++ {
		t := threads[i]
		conv[i] = &types.Thread{
			Uid: models.Uint32ToUid(t.Id),
		}

		// Set the first child node
		children, childCount := convertThreads(t.Children, conv[i])
		if len(children) > 0 {
			conv[i].FirstChild = children[0]
		}

		// Set the parent node
		if parent != nil {
			conv[i].Parent = parent

			// elements of threads are siblings
			if i > 0 {
				conv[i].PrevSibling = conv[i-1]
				conv[i-1].NextSibling = conv[i]
			}
		}

		count += childCount + 1
	}
	return conv, count
}
