package lib

import (
	"github.com/emersion/go-imap"

	"git.sr.ht/~sircmpwn/aerc2/worker/types"
)

type MessageStore struct {
	DirInfo  types.DirectoryInfo
	Messages map[uint32]*types.MessageInfo
	// Ordered list of known UIDs
	Uids []uint32
	// Map of uids we've asked the worker to fetch
	onUpdate       func(store *MessageStore)
	pendingBodies  map[uint32]interface{}
	pendingHeaders map[uint32]interface{}
	worker         *types.Worker
}

func NewMessageStore(worker *types.Worker,
	dirInfo *types.DirectoryInfo) *MessageStore {

	return &MessageStore{
		DirInfo: *dirInfo,

		pendingBodies:  make(map[uint32]interface{}),
		pendingHeaders: make(map[uint32]interface{}),
		worker:         worker,
	}
}

func (store *MessageStore) FetchHeaders(uids []uint32) {
	// TODO: this could be optimized by pre-allocating toFetch and trimming it
	// at the end. In practice we expect to get most messages back in one frame.
	var toFetch imap.SeqSet
	for _, uid := range uids {
		if _, ok := store.pendingHeaders[uid]; !ok {
			toFetch.AddNum(uint32(uid))
			store.pendingHeaders[uid] = nil
		}
	}
	if !toFetch.Empty() {
		store.worker.PostAction(&types.FetchMessageHeaders{
			Uids: toFetch,
		}, nil)
	}
}

func (store *MessageStore) Update(msg types.WorkerMessage) {
	update := false
	switch msg := msg.(type) {
	case *types.DirectoryInfo:
		store.DirInfo = *msg
		update = true
		break
	case *types.DirectoryContents:
		newMap := make(map[uint32]*types.MessageInfo)
		for _, uid := range msg.Uids {
			if msg, ok := store.Messages[uid]; ok {
				newMap[uid] = msg
			} else {
				newMap[uid] = nil
			}
		}
		store.Messages = newMap
		store.Uids = msg.Uids
		update = true
		break
	case *types.MessageInfo:
		// TODO: merge message info into existing record, if applicable
		store.Messages[msg.Uid] = msg
		if _, ok := store.pendingHeaders[msg.Uid]; msg.Envelope != nil && ok {
			delete(store.pendingHeaders, msg.Uid)
		}
		update = true
		break
	}
	if update && store.onUpdate != nil {
		store.onUpdate(store)
	}
}

func (store *MessageStore) OnUpdate(fn func(store *MessageStore)) {
	store.onUpdate = fn
}
