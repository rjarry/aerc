package imap

import (
	"github.com/emersion/go-imap"
	sortthread "github.com/emersion/go-imap-sortthread"

	"git.sr.ht/~sircmpwn/aerc/worker/types"
)

func (imapw *IMAPWorker) handleOpenDirectory(msg *types.OpenDirectory) {
	imapw.worker.Logger.Printf("Opening %s", msg.Directory)

	_, err := imapw.client.Select(msg.Directory, false)
	if err != nil {
		imapw.worker.PostMessage(&types.Error{
			Message: types.RespondTo(msg),
			Error:   err,
		}, nil)
	} else {
		imapw.worker.PostMessage(&types.Done{types.RespondTo(msg)}, nil)
	}
}

func (imapw *IMAPWorker) handleFetchDirectoryContents(
	msg *types.FetchDirectoryContents) {

	imapw.worker.Logger.Printf("Fetching UID list")

	seqSet := &imap.SeqSet{}
	seqSet.AddRange(1, imapw.selected.Messages)

	searchCriteria := &imap.SearchCriteria{
		SeqNum: seqSet,
	}
	sortCriteria := translateSortCriterions(msg.SortCriteria)

	var uids []uint32

	// If the server supports the SORT extension, do the sorting server side
	ok, err := imapw.client.sort.SupportSort()
	if err == nil && ok && len(sortCriteria) > 0 {
		uids, err = imapw.client.sort.UidSort(sortCriteria, searchCriteria)
		// copy in reverse as msgList displays backwards
		for i, j := 0, len(uids)-1; i < j; i, j = i+1, j-1 {
			uids[i], uids[j] = uids[j], uids[i]
		}
	} else {
		if err != nil {
			// Non fatal, but we do want to print to get some debug info
			imapw.worker.Logger.Printf("can't check for SORT support: %v", err)
		}
		uids, err = imapw.client.UidSearch(searchCriteria)
	}
	if err != nil {
		imapw.worker.PostMessage(&types.Error{
			Message: types.RespondTo(msg),
			Error:   err,
		}, nil)
	} else {
		imapw.worker.Logger.Printf("Found %d UIDs", len(uids))
		if len(imapw.seqMap) < len(uids) {
			imapw.seqMap = make([]uint32, len(uids))
		}
		imapw.worker.PostMessage(&types.DirectoryContents{
			Message: types.RespondTo(msg),
			Uids:    uids,
		}, nil)
		imapw.worker.PostMessage(&types.Done{types.RespondTo(msg)}, nil)
	}
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
	cs []*types.SortCriterion) []sortthread.SortCriterion {
	result := make([]sortthread.SortCriterion, 0, len(cs))
	for _, c := range cs {
		if f, ok := sortFieldMap[c.Field]; ok {
			result = append(result, sortthread.SortCriterion{f, c.Reverse})
		}
	}
	return result
}
