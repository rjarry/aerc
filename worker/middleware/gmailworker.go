package middleware

import (
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
	if msg, ok := msg.(*types.FetchMessageHeaders); ok && len(msg.Uids) > 0 {
		g.mu.Lock()

		handler := xgmext.NewHandler(g.client)
		uids, err := handler.FetchEntireThreads(msg.Uids)
		if err != nil {
			g.Errorf("failed to fetch entire threads: %v", err)
		}

		if len(uids) > 0 {
			msg.Uids = uids
		}

		g.mu.Unlock()
	}
	return g.WorkerInteractor.ProcessAction(msg)
}
