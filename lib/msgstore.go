package lib

import (
	"io"
	"sync"
	"time"

	"github.com/emersion/go-imap"

	"git.sr.ht/~sircmpwn/aerc2/worker/types"
)

// Accesses to fields must be guarded by MessageStore.Lock/Unlock
type MessageStore struct {
	sync.Mutex

	Deleted  map[uint32]interface{}
	DirInfo  types.DirectoryInfo
	Messages map[uint32]*types.MessageInfo
	// Ordered list of known UIDs
	Uids []uint32

	bodyCallbacks   map[uint32][]func(io.Reader)
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

		bodyCallbacks:   make(map[uint32][]func(io.Reader)),
		headerCallbacks: make(map[uint32][]func(*types.MessageInfo)),

		pendingBodies:  make(map[uint32]interface{}),
		pendingHeaders: make(map[uint32]interface{}),
		worker:         worker,
	}
}

func (store *MessageStore) FetchHeaders(uids []uint32,
	cb func(*types.MessageInfo)) {

	store.Lock()
	defer store.Unlock()

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

func (store *MessageStore) FetchFull(uids []uint32, cb func(io.Reader)) {
	store.Lock()
	defer store.Unlock()

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
					store.bodyCallbacks[uid] = []func(io.Reader){cb}
				}
			}
		}
	}
	if !toFetch.Empty() {
		store.worker.PostAction(&types.FetchFullMessages{Uids: toFetch}, nil)
	}
}

func (store *MessageStore) FetchBodyPart(
	uid uint32, part int, cb func(io.Reader)) {

	store.worker.PostAction(&types.FetchMessageBodyPart{
		Uid:  uid,
		Part: part,
	}, func(resp types.WorkerMessage) {
		msg, ok := resp.(*types.MessageBodyPart)
		if !ok {
			return
		}
		cb(msg.Reader)
	})
}

func merge(to *types.MessageInfo, from *types.MessageInfo) {

	if from.BodyStructure != nil {
		to.BodyStructure = from.BodyStructure
	}
	if from.Envelope != nil {
		to.Envelope = from.Envelope
	}
	if len(from.Flags) != 0 {
		to.Flags = from.Flags
	}
	if from.Size != 0 {
		to.Size = from.Size
	}
	var zero time.Time
	if from.InternalDate != zero {
		to.InternalDate = from.InternalDate
	}
}

func (store *MessageStore) Update(msg types.WorkerMessage) {
	store.Lock()

	update := false
	switch msg := msg.(type) {
	case *types.DirectoryInfo:
		store.DirInfo = *msg
		if store.DirInfo.Exists != len(store.Uids) {
			store.worker.PostAction(&types.FetchDirectoryContents{}, nil)
		}
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
			merge(existing, msg)
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
	case *types.FullMessage:
		if _, ok := store.pendingBodies[msg.Uid]; ok {
			delete(store.pendingBodies, msg.Uid)
			if cbs, ok := store.bodyCallbacks[msg.Uid]; ok {
				for _, cb := range cbs {
					cb(msg.Reader)
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
		for _, uid := range store.Uids {
			if _, deleted := toDelete[uid]; !deleted && j < len(uids) {
				uids[j] = uid
				j += 1
			}
		}
		store.Uids = uids
		update = true
	}

	store.Unlock()

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

func (store *MessageStore) Delete(uids []uint32,
	cb func(msg types.WorkerMessage)) {
	store.Lock()

	var set imap.SeqSet
	for _, uid := range uids {
		set.AddNum(uid)
		store.Deleted[uid] = nil
	}

	store.Unlock()

	store.worker.PostAction(&types.DeleteMessages{Uids: set}, cb)
	store.update()
}

func (store *MessageStore) Copy(uids []uint32, dest string,
	cb func(msg types.WorkerMessage)) {
	var set imap.SeqSet
	for _, uid := range uids {
		set.AddNum(uid)
	}

	store.worker.PostAction(&types.CopyMessages{
		Destination: dest,
		Uids:        set,
	}, cb)
}

func (store *MessageStore) Move(uids []uint32, dest string,
	cb func(msg types.WorkerMessage)) {
	store.Lock()

	var set imap.SeqSet
	for _, uid := range uids {
		set.AddNum(uid)
		store.Deleted[uid] = nil
	}

	store.Unlock()

	store.worker.PostAction(&types.CopyMessages{
		Destination: dest,
		Uids:        set,
	}, func(msg types.WorkerMessage) {
		switch msg.(type) {
		case *types.Error:
			cb(msg)
		case *types.Done:
			store.worker.PostAction(&types.DeleteMessages{Uids: set}, cb)
		}
	})

	store.update()
}
