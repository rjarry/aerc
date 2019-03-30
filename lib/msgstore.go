package lib

import (
	"github.com/emersion/go-imap"
	"github.com/mohamedattahri/mail"

	"git.sr.ht/~sircmpwn/aerc2/worker/types"
)

type MessageStore struct {
	Deleted  map[uint32]interface{}
	DirInfo  types.DirectoryInfo
	Messages map[uint32]*types.MessageInfo
	// Ordered list of known UIDs
	Uids []uint32

	bodyCallbacks   map[uint32][]func(*mail.Message)
	headerCallbacks map[uint32][]func(*types.MessageInfo)

	// Map of uids we've asked the worker to fetch
	onUpdate       func(store *MessageStore) // TODO: multiple onUpdate handlers
	pendingBodies  map[uint32]interface{}
	pendingHeaders map[uint32]interface{}
	worker         *types.Worker
}

func NewMessageStore(worker *types.Worker,
	dirInfo *types.DirectoryInfo) *MessageStore {

	return &MessageStore{
		Deleted: make(map[uint32]interface{}),
		DirInfo: *dirInfo,

		bodyCallbacks:   make(map[uint32][]func(*mail.Message)),
		headerCallbacks: make(map[uint32][]func(*types.MessageInfo)),

		pendingBodies:  make(map[uint32]interface{}),
		pendingHeaders: make(map[uint32]interface{}),
		worker:         worker,
	}
}

func (store *MessageStore) FetchHeaders(uids []uint32,
	cb func(*types.MessageInfo)) {

	// TODO: this could be optimized by pre-allocating toFetch and trimming it
	// at the end. In practice we expect to get most messages back in one frame.
	var toFetch imap.SeqSet
	for _, uid := range uids {
		if _, ok := store.pendingHeaders[uid]; !ok {
			toFetch.AddNum(uint32(uid))
			store.pendingHeaders[uid] = nil
			if cb != nil {
				if list, ok := store.headerCallbacks[uid]; ok {
					store.headerCallbacks[uid] = append(list, cb)
				} else {
					store.headerCallbacks[uid] = []func(*types.MessageInfo){cb}
				}
			}
		}
	}
	if !toFetch.Empty() {
		store.worker.PostAction(&types.FetchMessageHeaders{Uids: toFetch}, nil)
	}
}

func (store *MessageStore) FetchBodies(uids []uint32,
	cb func(*mail.Message)) {

	// TODO: this could be optimized by pre-allocating toFetch and trimming it
	// at the end. In practice we expect to get most messages back in one frame.
	var toFetch imap.SeqSet
	for _, uid := range uids {
		if _, ok := store.pendingBodies[uid]; !ok {
			toFetch.AddNum(uint32(uid))
			store.pendingBodies[uid] = nil
			if cb != nil {
				if list, ok := store.bodyCallbacks[uid]; ok {
					store.bodyCallbacks[uid] = append(list, cb)
				} else {
					store.bodyCallbacks[uid] = []func(*mail.Message){cb}
				}
			}
		}
	}
	if !toFetch.Empty() {
		store.worker.PostAction(&types.FetchMessageBodies{Uids: toFetch}, nil)
	}
}

func (store *MessageStore) merge(
	to *types.MessageInfo, from *types.MessageInfo) {

	// TODO: Merge more shit
	if from.Envelope != nil {
		to.Envelope = from.Envelope
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
		if existing, ok := store.Messages[msg.Uid]; ok && existing != nil {
			store.merge(existing, msg)
		} else {
			store.Messages[msg.Uid] = msg
		}
		if _, ok := store.pendingHeaders[msg.Uid]; msg.Envelope != nil && ok {
			delete(store.pendingHeaders, msg.Uid)
			if cbs, ok := store.headerCallbacks[msg.Uid]; ok {
				for _, cb := range cbs {
					cb(msg)
				}
			}
		}
		update = true
	case *types.MessageBody:
		if _, ok := store.pendingBodies[msg.Uid]; ok {
			delete(store.pendingBodies, msg.Uid)
			if cbs, ok := store.bodyCallbacks[msg.Uid]; ok {
				for _, cb := range cbs {
					cb(msg.Mail)
				}
			}
		}
	case *types.MessagesDeleted:
		toDelete := make(map[uint32]interface{})
		for _, uid := range msg.Uids {
			toDelete[uid] = nil
			delete(store.Messages, uid)
			if _, ok := store.Deleted[uid]; ok {
				delete(store.Deleted, uid)
			}
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
	if update {
		store.update()
	}
}

func (store *MessageStore) OnUpdate(fn func(store *MessageStore)) {
	store.onUpdate = fn
}

func (store *MessageStore) update() {
	if store.onUpdate != nil {
		store.onUpdate(store)
	}
}

func (store *MessageStore) Delete(uids []uint32) {
	var set imap.SeqSet
	for _, uid := range uids {
		set.AddNum(uid)
		store.Deleted[uid] = nil
	}
	store.worker.PostAction(&types.DeleteMessages{Uids: set}, nil)
	store.update()
}
