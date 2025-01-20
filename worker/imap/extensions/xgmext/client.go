package xgmext

import (
	"errors"
	"fmt"

	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/models"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-imap/commands"
	"github.com/emersion/go-imap/responses"
)

type handler struct {
	client *client.Client
}

func NewHandler(c *client.Client) *handler {
	return &handler{client: c}
}

func (h handler) FetchEntireThreads(requested []models.UID) ([]models.UID, error) {
	threadIds, err := h.fetchThreadIds(requested)
	if err != nil {
		return nil,
			fmt.Errorf("failed to fetch thread IDs: %w", err)
	}
	uids, err := h.searchUids(threadIds)
	if err != nil {
		return nil,
			fmt.Errorf("failed to search for thread IDs: %w", err)
	}
	return uids, nil
}

func (h handler) fetchThreadIds(uids []models.UID) ([]string, error) {
	messages := make(chan *imap.Message)
	done := make(chan error)

	thriditem := imap.FetchItem("X-GM-THRID")
	items := []imap.FetchItem{
		thriditem,
	}

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
		set.AddNum(models.UidToUint32(uid))
	}
	err := h.client.UidFetch(&set, items, messages)
	<-done

	thrid := make([]string, 0, len(m))
	for id := range m {
		thrid = append(thrid, id)
	}
	return thrid, err
}

func (h handler) searchUids(thrid []string) ([]models.UID, error) {
	if len(thrid) == 0 {
		return nil, errors.New("no thread IDs provided")
	}
	return h.runSearch(NewThreadIDSearch(thrid))
}

func (h handler) RawSearch(rawSearch string) ([]models.UID, error) {
	return h.runSearch(NewRawSearch(rawSearch))
}

func (h handler) runSearch(cmd imap.Commander) ([]models.UID, error) {
	if h.client.State() != imap.SelectedState {
		return nil, errors.New("no mailbox selected")
	}
	cmd = &commands.Uid{Cmd: cmd}
	res := new(responses.Search)
	status, err := h.client.Execute(cmd, res)
	if err != nil {
		return nil, fmt.Errorf("imap execute failed: %w", err)
	}
	var uids []models.UID
	for _, i := range res.Ids {
		uids = append(uids, models.Uint32ToUid(i))
	}
	return uids, status.Err()
}
