package lib

import (
	"io"
	"time"

	"github.com/emersion/go-imap"

	"git.sr.ht/~sircmpwn/aerc/models"
	"git.sr.ht/~sircmpwn/aerc/worker/types"
)

// Accesses to fields must be guarded by MessageStore.Lock/Unlock
type MessageStore struct {
	Deleted  map[uint32]interface{}
	DirInfo  models.DirectoryInfo
	Messages map[uint32]*models.MessageInfo
	// Ordered list of known UIDs
	Uids []uint32

	selected        int
	bodyCallbacks   map[uint32][]func(io.Reader)
	headerCallbacks map[uint32][]func(*types.MessageInfo)

	// Search/filter results
	results     []uint32
	resultIndex int

	// Map of uids we've asked the worker to fetch
	onUpdate       func(store *MessageStore) // TODO: multiple onUpdate handlers
	pendingBodies  map[uint32]interface{}
	pendingHeaders map[uint32]interface{}
	worker         *types.Worker
}

func NewMessageStore(worker *types.Worker,
	dirInfo *models.DirectoryInfo) *MessageStore {

	return &MessageStore{
		Deleted: make(map[uint32]interface{}),
		DirInfo: *dirInfo,

		selected:        0,
		bodyCallbacks:   make(map[uint32][]func(io.Reader)),
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
	var toFetch []uint32
	for _, uid := range uids {
		if _, ok := store.pendingHeaders[uid]; !ok {
			toFetch = append(toFetch, uid)
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
	if len(toFetch) > 0 {
		store.worker.PostAction(&types.FetchMessageHeaders{Uids: toFetch}, nil)
	}
}

func (store *MessageStore) FetchFull(uids []uint32, cb func(io.Reader)) {
	// TODO: this could be optimized by pre-allocating toFetch and trimming it
	// at the end. In practice we expect to get most messages back in one frame.
	var toFetch []uint32
	for _, uid := range uids {
		if _, ok := store.pendingBodies[uid]; !ok {
			toFetch = append(toFetch, uid)
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
	if len(toFetch) > 0 {
		store.worker.PostAction(&types.FetchFullMessages{Uids: toFetch}, nil)
	}
}

func (store *MessageStore) FetchBodyPart(
	uid uint32, part []int, cb func(io.Reader)) {

	store.worker.PostAction(&types.FetchMessageBodyPart{
		Uid:  uid,
		Part: part,
	}, func(resp types.WorkerMessage) {
		msg, ok := resp.(*types.MessageBodyPart)
		if !ok {
			return
		}
		cb(msg.Part.Reader)
	})
}

func merge(to *models.MessageInfo, from *models.MessageInfo) {
	if from.BodyStructure != nil {
		to.BodyStructure = from.BodyStructure
	}
	if from.Envelope != nil {
		to.Envelope = from.Envelope
	}
	to.Flags = from.Flags
	if from.Size != 0 {
		to.Size = from.Size
	}
	var zero time.Time
	if from.InternalDate != zero {
		to.InternalDate = from.InternalDate
	}
}

func (store *MessageStore) Update(msg types.WorkerMessage) {
	update := false
	switch msg := msg.(type) {
	case *types.DirectoryInfo:
		store.DirInfo = *msg.Info
		store.worker.PostAction(&types.FetchDirectoryContents{}, nil)
		update = true
	case *types.DirectoryContents:
		newMap := make(map[uint32]*models.MessageInfo)
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
		if existing, ok := store.Messages[msg.Info.Uid]; ok && existing != nil {
			merge(existing, msg.Info)
		} else {
			store.Messages[msg.Info.Uid] = msg.Info
		}
		if _, ok := store.pendingHeaders[msg.Info.Uid]; msg.Info.Envelope != nil && ok {
			delete(store.pendingHeaders, msg.Info.Uid)
			if cbs, ok := store.headerCallbacks[msg.Info.Uid]; ok {
				for _, cb := range cbs {
					cb(msg)
				}
			}
		}
		update = true
	case *types.FullMessage:
		if _, ok := store.pendingBodies[msg.Content.Uid]; ok {
			delete(store.pendingBodies, msg.Content.Uid)
			if cbs, ok := store.bodyCallbacks[msg.Content.Uid]; ok {
				for _, cb := range cbs {
					cb(msg.Content.Reader)
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

	for _, uid := range uids {
		store.Deleted[uid] = nil
	}

	store.worker.PostAction(&types.DeleteMessages{Uids: uids}, cb)
	store.update()
}

func (store *MessageStore) Copy(uids []uint32, dest string, createDest bool,
	cb func(msg types.WorkerMessage)) {

	if createDest {
		store.worker.PostAction(&types.CreateDirectory{
			Directory: dest,
		}, cb)
	}

	store.worker.PostAction(&types.CopyMessages{
		Destination: dest,
		Uids:        uids,
	}, cb)
}

func (store *MessageStore) Move(uids []uint32, dest string, createDest bool,
	cb func(msg types.WorkerMessage)) {

	for _, uid := range uids {
		store.Deleted[uid] = nil
	}

	if createDest {
		store.worker.PostAction(&types.CreateDirectory{
			Directory: dest,
		}, cb)
	}

	store.worker.PostAction(&types.CopyMessages{
		Destination: dest,
		Uids:        uids,
	}, func(msg types.WorkerMessage) {
		switch msg.(type) {
		case *types.Error:
			cb(msg)
		case *types.Done:
			store.worker.PostAction(&types.DeleteMessages{Uids: uids}, cb)
		}
	})

	store.update()
}

func (store *MessageStore) Read(uids []uint32, read bool,
	cb func(msg types.WorkerMessage)) {

	store.worker.PostAction(&types.ReadMessages{
		Read: read,
		Uids: uids,
	}, cb)
}

func (store *MessageStore) Selected() *models.MessageInfo {
	return store.Messages[store.Uids[len(store.Uids)-store.selected-1]]
}

func (store *MessageStore) SelectedIndex() int {
	return store.selected
}

func (store *MessageStore) Select(index int) {
	store.selected = index
	for ; store.selected < 0; store.selected = len(store.Uids) + store.selected {
		/* This space deliberately left blank */
	}
	if store.selected > len(store.Uids) {
		store.selected = len(store.Uids)
	}
}

func (store *MessageStore) nextPrev(delta int) {
	if len(store.Uids) == 0 {
		return
	}
	store.selected += delta
	if store.selected < 0 {
		store.selected = 0
	}
	if store.selected >= len(store.Uids) {
		store.selected = len(store.Uids) - 1
	}
}

func (store *MessageStore) Next() {
	store.nextPrev(1)
}

func (store *MessageStore) Prev() {
	store.nextPrev(-1)
}

func (store *MessageStore) Search(c *imap.SearchCriteria, cb func([]uint32)) {
	store.worker.PostAction(&types.SearchDirectory{
		Criteria: c,
	}, func(msg types.WorkerMessage) {
		switch msg := msg.(type) {
		case *types.SearchResults:
			cb(msg.Uids)
		}
	})
}

func (store *MessageStore) ApplySearch(results []uint32) {
	store.results = results
	store.resultIndex = -1
	store.NextResult()
}

func (store *MessageStore) nextPrevResult(delta int) {
	if len(store.results) == 0 {
		return
	}
	store.resultIndex += delta
	if store.resultIndex >= len(store.results) {
		store.resultIndex = 0
	}
	if store.resultIndex < 0 {
		store.resultIndex = len(store.results) - 1
	}
	for i, uid := range store.Uids {
		if store.results[len(store.results)-store.resultIndex-1] == uid {
			store.Select(len(store.Uids) - i - 1)
			break
		}
	}
	store.update()
}

func (store *MessageStore) NextResult() {
	store.nextPrevResult(1)
}

func (store *MessageStore) PrevResult() {
	store.nextPrevResult(-1)
}
