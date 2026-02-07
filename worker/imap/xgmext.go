package imap

import (
	"strconv"
	"strings"

	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

// handleGmailFilter handles FetchDirectoryContents with Gmail X-GM-RAW filtering.
// Returns true if the request was handled (caller should not proceed with normal filtering).
func (w *IMAPWorker) handleGmailFilter(msg *types.FetchDirectoryContents) bool {
	if msg.Filter == nil || len(msg.Filter.Terms) == 0 {
		return false
	}
	if !msg.Filter.UseExtension {
		w.worker.Debugf("use regular imap filter instead of X-GM-EXT1: extension flag not set")
		return false
	}

	search := strings.Join(msg.Filter.Terms, " ")
	w.worker.Debugf("X-GM-EXT1 filter term: '%s'", search)

	uids, err := w.client.xgmext.RawSearch(strconv.Quote(search))
	if err != nil {
		w.worker.Errorf("X-GM-EXT1 filter failed: %v", err)
		w.worker.Warnf("falling back to imap filtering")
		return false
	}

	w.worker.PostMessage(&types.DirectoryContents{
		Message: types.RespondTo(msg),
		Uids:    w.Uint32ToUidList(uids),
	}, nil)

	return true
}

// handleGmailSearch handles SearchDirectory with Gmail X-GM-RAW searching.
// Returns true if the request was handled (caller should not proceed with normal search).
func (w *IMAPWorker) handleGmailSearch(msg *types.SearchDirectory) bool {
	if msg.Criteria == nil || len(msg.Criteria.Terms) == 0 {
		return false
	}
	if !msg.Criteria.UseExtension {
		w.worker.Debugf("use regular imap search instead of X-GM-EXT1: extension flag not set")
		return false
	}

	search := strings.Join(msg.Criteria.Terms, " ")
	w.worker.Debugf("X-GM-EXT1 search term: '%s'", search)

	uids, err := w.client.xgmext.RawSearch(strconv.Quote(search))
	if err != nil {
		w.worker.Errorf("X-GM-EXT1 search failed: %v", err)
		w.worker.Warnf("falling back to regular imap search.")
		return false
	}

	w.worker.PostMessage(&types.SearchResults{
		Message: types.RespondTo(msg),
		Uids:    w.Uint32ToUidList(uids),
	}, nil)

	return true
}

// fetchEntireThreads fetches all message UIDs that belong to the same threads
// as the requested UIDs using the X-GM-THRID extension.
func (w *IMAPWorker) fetchEntireThreads(requested []models.UID) ([]models.UID, error) {
	uids, err := w.client.xgmext.FetchEntireThreads(w.UidToUint32List(requested))
	if err != nil {
		return nil, err
	}
	return w.Uint32ToUidList(uids), nil
}
