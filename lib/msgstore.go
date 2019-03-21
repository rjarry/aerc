package lib

import (
	"fmt"

	"github.com/emersion/go-imap"

	"git.sr.ht/~sircmpwn/aerc2/worker/types"
)

type MessageStore struct {
	DirInfo  types.DirectoryInfo
	Messages map[uint32]*types.MessageInfo
	// Ordered list of known UIDs
	Uids []uint32
	// Map of uids we've asked the worker to fetch
	onUpdate       func(store *MessageStore) // TODO: multiple onUpdate handlers
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
	case *types.MessageInfo:
		// TODO: merge message info into existing record, if applicable
		store.Messages[msg.Uid] = msg
		if _, ok := store.pendingHeaders[msg.Uid]; msg.Envelope != nil && ok {
			delete(store.pendingHeaders, msg.Uid)
		}
		update = true
	case *types.MessagesDeleted:
		toDelete := make(map[uint32]interface{})
		for _, uid := range msg.Uids {
			toDelete[uid] = nil
			delete(store.Messages, uid)
		}
		uids := make([]uint32, len(store.Uids)-len(msg.Uids))
		j := 0
		for i, uid := range store.Uids {
			if _, deleted := toDelete[uid]; !deleted {
				uids[j] = store.Uids[i]
				j += 1
			}
		}
		store.Uids = uids
		update = true
	}
	if update && store.onUpdate != nil {
		store.onUpdate(store)
	}
}

func (store *MessageStore) OnUpdate(fn func(store *MessageStore)) {
	store.onUpdate = fn
}

func (store *MessageStore) Delete(uids []uint32) {
	var set imap.SeqSet
	for _, uid := range uids {
		set.AddNum(uid)
	}
	store.worker.PostAction(&types.DeleteMessages{Uids: set}, nil)
}
