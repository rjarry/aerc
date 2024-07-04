package middleware

import (
	"strconv"
	"strings"
	"sync"

	"git.sr.ht/~rjarry/aerc/worker/imap/extensions/xgmext"
	"git.sr.ht/~rjarry/aerc/worker/types"
	"github.com/emersion/go-imap/client"
)

type gmailWorker struct {
	types.WorkerInteractor
	mu     sync.Mutex
	client *client.Client
}

// NewGmailWorker returns an IMAP middleware for the X-GM-EXT-1 extension
func NewGmailWorker(base types.WorkerInteractor, c *client.Client,
) types.WorkerInteractor {
	base.Infof("loading worker middleware: X-GM-EXT-1")

	// avoid double wrapping; unwrap and check for another gmail handler
	for iter := base; iter != nil; iter = iter.Unwrap() {
		if g, ok := iter.(*gmailWorker); ok {
			base.Infof("already loaded; resetting")
			err := g.reset(c)
			if err != nil {
				base.Errorf("reset failed: %v", err)
			}
			return base
		}
	}
	return &gmailWorker{WorkerInteractor: base, client: c}
}

func (g *gmailWorker) Unwrap() types.WorkerInteractor {
	return g.WorkerInteractor
}

func (g *gmailWorker) reset(c *client.Client) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.client = c
	return nil
}

func (g *gmailWorker) ProcessAction(msg types.WorkerMessage) types.WorkerMessage {
	switch msg := msg.(type) {
	case *types.FetchMessageHeaders:
		handler := xgmext.NewHandler(g.client)

		g.mu.Lock()
		uids, err := handler.FetchEntireThreads(msg.Uids)
		g.mu.Unlock()
		if err != nil {
			g.Warnf("failed to fetch entire threads: %v", err)
		}

		if len(uids) > 0 {
			msg.Uids = uids
		}

	case *types.FetchDirectoryContents:
		if msg.Filter == nil || (msg.Filter != nil &&
			len(msg.Filter.Terms) == 0) {
			break
		}
		if !msg.Filter.UseExtension {
			g.Debugf("use regular imap filter instead of X-GM-EXT1: " +
				"extension flag not set")
			break
		}

		search := strings.Join(msg.Filter.Terms, " ")
		g.Debugf("X-GM-EXT1 filter term: '%s'", search)

		handler := xgmext.NewHandler(g.client)

		g.mu.Lock()
		uids, err := handler.RawSearch(strconv.Quote(search))
		g.mu.Unlock()
		if err != nil {
			g.Errorf("X-GM-EXT1 filter failed: %v", err)
			g.Warnf("falling back to imap filtering")
			break
		}

		g.PostMessage(&types.DirectoryContents{
			Message: types.RespondTo(msg),
			Uids:    uids,
		}, nil)

		g.PostMessage(&types.Done{Message: types.RespondTo(msg)}, nil)

		return &types.Unsupported{}

	case *types.SearchDirectory:
		if msg.Criteria == nil || (msg.Criteria != nil &&
			len(msg.Criteria.Terms) == 0) {
			break
		}
		if !msg.Criteria.UseExtension {
			g.Debugf("use regular imap search instead of X-GM-EXT1: " +
				"extension flag not set")
			break
		}

		search := strings.Join(msg.Criteria.Terms, " ")
		g.Debugf("X-GM-EXT1 search term: '%s'", search)
		handler := xgmext.NewHandler(g.client)

		g.mu.Lock()
		uids, err := handler.RawSearch(strconv.Quote(search))
		g.mu.Unlock()
		if err != nil {
			g.Errorf("X-GM-EXT1 search failed: %v", err)
			g.Warnf("falling back to regular imap search.")
			break
		}

		g.PostMessage(&types.SearchResults{
			Message: types.RespondTo(msg),
			Uids:    uids,
		}, nil)

		g.PostMessage(&types.Done{Message: types.RespondTo(msg)}, nil)

		return &types.Unsupported{}
	}
	return g.WorkerInteractor.ProcessAction(msg)
}
