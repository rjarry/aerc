package xgmext

import (
	"errors"
	"fmt"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-imap/commands"
	"github.com/emersion/go-imap/responses"

	"git.sr.ht/~rjarry/aerc/lib/log"
)

// XGMExtClient is a client for the X-GM-EXT-1 Gmail extension.
type XGMExtClient struct {
	c *client.Client
}

func NewXGMExtClient(c *client.Client) *XGMExtClient {
	return &XGMExtClient{c: c}
}

func (x *XGMExtClient) FetchEntireThreads(requested []uint32) ([]uint32, error) {
	threadIds, err := x.fetchThreadIds(requested)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch thread IDs: %w", err)
	}
	if len(threadIds) == 0 {
		return nil, errors.New("no thread IDs provided")
	}
	uids, err := x.runSearch(NewThreadIDSearch(threadIds))
	if err != nil {
		return nil, fmt.Errorf("failed to search for thread IDs: %w", err)
	}
	return uids, nil
}

func (x *XGMExtClient) fetchThreadIds(uids []uint32) ([]string, error) {
	messages := make(chan *imap.Message)
	done := make(chan error)

	thriditem := imap.FetchItem("X-GM-THRID")
	items := []imap.FetchItem{thriditem}

	m := make(map[string]struct{}, len(uids))
	go func() {
		defer log.PanicHandler()
		for msg := range messages {
			if msg == nil {
				continue
			}
			item, ok := msg.Items[thriditem].(string)
			if ok {
				m[item] = struct{}{}
			}
		}
		done <- nil
	}()

	var set imap.SeqSet
	for _, uid := range uids {
		set.AddNum(uid)
	}
	err := x.c.UidFetch(&set, items, messages)
	<-done

	thrid := make([]string, 0, len(m))
	for id := range m {
		thrid = append(thrid, id)
	}
	return thrid, err
}

func (x *XGMExtClient) RawSearch(rawSearch string) ([]uint32, error) {
	return x.runSearch(NewRawSearch(rawSearch))
}

func (x *XGMExtClient) runSearch(cmd imap.Commander) ([]uint32, error) {
	if x.c.State() != imap.SelectedState {
		return nil, errors.New("no mailbox selected")
	}
	cmd = &commands.Uid{Cmd: cmd}
	res := new(responses.Search)
	status, err := x.c.Execute(cmd, res)
	if err != nil {
		return nil, fmt.Errorf("imap execute failed: %w", err)
	}
	return res.Ids, status.Err()
}
