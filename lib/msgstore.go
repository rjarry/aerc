package lib

import (
	"io"
	"time"

	"git.sr.ht/~sircmpwn/aerc/lib/sort"
	"git.sr.ht/~sircmpwn/aerc/models"
	"git.sr.ht/~sircmpwn/aerc/worker/types"
)

// Accesses to fields must be guarded by MessageStore.Lock/Unlock
type MessageStore struct {
	Deleted  map[uint32]interface{}
	DirInfo  models.DirectoryInfo
	Messages map[uint32]*models.MessageInfo
	// Ordered list of known UIDs
	uids []uint32

	selected        int
	bodyCallbacks   map[uint32][]func(io.Reader)
	headerCallbacks map[uint32][]func(*types.MessageInfo)

	//marking
	marked         map[uint32]struct{}
	visualStartUid uint32
	visualMarkMode bool

	// Search/filter results
	results     []uint32
	resultIndex int
	filter      bool

	defaultSortCriteria []*types.SortCriterion

	// Map of uids we've asked the worker to fetch
	onUpdate       func(store *MessageStore) // TODO: multiple onUpdate handlers
	onUpdateDirs   func()
	pendingBodies  map[uint32]interface{}
	pendingHeaders map[uint32]interface{}
	worker         *types.Worker

	triggerNewEmail        func(*models.MessageInfo)
	triggerDirectoryChange func()
}

func NewMessageStore(worker *types.Worker,
	dirInfo *models.DirectoryInfo,
	defaultSortCriteria []*types.SortCriterion,
	triggerNewEmail func(*models.MessageInfo),
	triggerDirectoryChange func()) *MessageStore {

	return &MessageStore{
		Deleted:  make(map[uint32]interface{}),
		DirInfo:  *dirInfo,
		Messages: make(map[uint32]*models.MessageInfo),

		selected:        0,
		marked:          make(map[uint32]struct{}),
		bodyCallbacks:   make(map[uint32][]func(io.Reader)),
		headerCallbacks: make(map[uint32][]func(*types.MessageInfo)),

		defaultSortCriteria: defaultSortCriteria,

		pendingBodies:  make(map[uint32]interface{}),
		pendingHeaders: make(map[uint32]interface{}),
		worker:         worker,

		triggerNewEmail:        triggerNewEmail,
		triggerDirectoryChange: triggerDirectoryChange,
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
		store.worker.PostAction(&types.FetchFullMessages{
			Uids: toFetch,
		}, func(msg types.WorkerMessage) {
			switch msg.(type) {
			case *types.Error:
				for _, uid := range toFetch {
					delete(store.bodyCallbacks, uid)
				}
			}
		})
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
	directoryChange := false
	switch msg := msg.(type) {
	case *types.DirectoryInfo:
		store.DirInfo = *msg.Info
		store.worker.PostAction(&types.FetchDirectoryContents{
			SortCriteria: store.defaultSortCriteria,
		}, nil)
		update = true
	case *types.DirectoryContents:
		newMap := make(map[uint32]*models.MessageInfo)
		for _, uid := range msg.Uids {
			if msg, ok := store.Messages[uid]; ok {
				newMap[uid] = msg
			} else {
				newMap[uid] = nil
				directoryChange = true
			}
		}
		store.Messages = newMap
		store.uids = msg.Uids
		update = true
	case *types.MessageInfo:
		if existing, ok := store.Messages[msg.Info.Uid]; ok && existing != nil {
			merge(existing, msg.Info)
		} else {
			store.Messages[msg.Info.Uid] = msg.Info
		}
		seen := false
		recent := false
		for _, flag := range msg.Info.Flags {
			if flag == models.RecentFlag {
				recent = true
			} else if flag == models.SeenFlag {
				seen = true
			}
		}
		if !seen && recent {
			store.triggerNewEmail(msg.Info)
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
				delete(store.bodyCallbacks, msg.Content.Uid)
			}
		}
	case *types.MessagesDeleted:
		if len(store.uids) < len(msg.Uids) {
			update = true
			break
		}

		toDelete := make(map[uint32]interface{})
		for _, uid := range msg.Uids {
			toDelete[uid] = nil
			delete(store.Messages, uid)
			delete(store.Deleted, uid)
			delete(store.marked, uid)
		}
		uids := make([]uint32, len(store.uids)-len(msg.Uids))
		j := 0
		for _, uid := range store.uids {
			if _, deleted := toDelete[uid]; !deleted && j < len(uids) {
				uids[j] = uid
				j += 1
			}
		}
		store.uids = uids
		update = true
	}

	if update {
		store.update()
	}

	if directoryChange && store.triggerDirectoryChange != nil {
		store.triggerDirectoryChange()
	}
}

func (store *MessageStore) OnUpdate(fn func(store *MessageStore)) {
	store.onUpdate = fn
}

func (store *MessageStore) OnUpdateDirs(fn func()) {
	store.onUpdateDirs = fn
}

func (store *MessageStore) update() {
	if store.onUpdate != nil {
		store.onUpdate(store)
	}
	if store.onUpdateDirs != nil {
		store.onUpdateDirs()
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
			Quiet:     true,
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
			Quiet:     true,
		}, nil) // quiet doesn't return an error, don't want the done cb here
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

func (store *MessageStore) Uids() []uint32 {
	if store.filter {
		return store.results
	}
	return store.uids
}

func (store *MessageStore) Selected() *models.MessageInfo {
	return store.Messages[store.Uids()[len(store.Uids())-store.selected-1]]
}

func (store *MessageStore) SelectedIndex() int {
	return store.selected
}

func (store *MessageStore) Select(index int) {
	uids := store.Uids()
	store.selected = index
	if store.selected < 0 {
		store.selected = len(uids) - 1
	} else if store.selected > len(uids) {
		store.selected = len(uids)
	}
	store.updateVisual()
}

// Mark sets the marked state on a MessageInfo
func (store *MessageStore) Mark(uid uint32) {
	if store.visualMarkMode {
		// visual mode has override, bogus input from user
		return
	}
	store.marked[uid] = struct{}{}
}

// Unmark removes the marked state on a MessageInfo
func (store *MessageStore) Unmark(uid uint32) {
	if store.visualMarkMode {
		// user probably wanted to clear the visual marking
		store.ClearVisualMark()
		return
	}
	delete(store.marked, uid)
}

// ToggleMark toggles the marked state on a MessageInfo
func (store *MessageStore) ToggleMark(uid uint32) {
	if store.visualMarkMode {
		// visual mode has override, bogus input from user
		return
	}
	if store.IsMarked(uid) {
		store.Unmark(uid)
	} else {
		store.Mark(uid)
	}
}

// resetMark removes the marking from all messages
func (store *MessageStore) resetMark() {
	store.marked = make(map[uint32]struct{})
}

//IsMarked checks whether a MessageInfo has been marked
func (store *MessageStore) IsMarked(uid uint32) bool {
	_, marked := store.marked[uid]
	return marked
}

//ToggleVisualMark enters or leaves the visual marking mode
func (store *MessageStore) ToggleVisualMark() {
	store.visualMarkMode = !store.visualMarkMode
	switch store.visualMarkMode {
	case true:
		// just entered visual mode, reset whatever marking was already done
		store.resetMark()
		store.visualStartUid = store.Selected().Uid
		store.marked[store.visualStartUid] = struct{}{}
	case false:
		// visual mode ended, nothing to do
		return
	}
}

//ClearVisualMark leaves the visual marking mode and resets any marking
func (store *MessageStore) ClearVisualMark() {
	store.resetMark()
	store.visualMarkMode = false
	store.visualStartUid = 0
}

// Marked returns the uids of all marked messages
func (store *MessageStore) Marked() []uint32 {
	marked := make([]uint32, len(store.marked))
	i := 0
	for uid := range store.marked {
		marked[i] = uid
		i++
	}
	return marked
}

func (store *MessageStore) updateVisual() {
	if !store.visualMarkMode {
		// nothing to do
		return
	}
	startIdx := store.visualStartIdx()
	if startIdx < 0 {
		// something deleted the startuid, abort the marking process
		store.ClearVisualMark()
		return
	}
	uidLen := len(store.Uids())
	// store.selected is the inverted form of the actual array
	selectedIdx := uidLen - store.selected - 1
	var visUids []uint32
	if selectedIdx > startIdx {
		visUids = store.Uids()[startIdx : selectedIdx+1]
	} else {
		visUids = store.Uids()[selectedIdx : startIdx+1]
	}
	store.resetMark()
	for _, uid := range visUids {
		store.marked[uid] = struct{}{}
	}
}

func (store *MessageStore) NextPrev(delta int) {
	uids := store.Uids()
	if len(uids) == 0 {
		return
	}
	store.selected += delta
	if store.selected < 0 {
		store.selected = 0
	}
	if store.selected >= len(uids) {
		store.selected = len(uids) - 1
	}
	store.updateVisual()
	nextResultIndex := len(store.results) - store.resultIndex - 2*delta
	if nextResultIndex < 0 || nextResultIndex >= len(store.results) {
		return
	}
	nextResultUid := store.results[nextResultIndex]
	selectedUid := uids[len(uids)-store.selected-1]
	if nextResultUid == selectedUid {
		store.resultIndex += delta
	}
}

func (store *MessageStore) Next() {
	store.NextPrev(1)
}

func (store *MessageStore) Prev() {
	store.NextPrev(-1)
}

func (store *MessageStore) Search(args []string, cb func([]uint32)) {
	store.worker.PostAction(&types.SearchDirectory{
		Argv: args,
	}, func(msg types.WorkerMessage) {
		switch msg := msg.(type) {
		case *types.SearchResults:
			sort.SortBy(msg.Uids, store.uids)
			cb(msg.Uids)
		}
	})
}

func (store *MessageStore) ApplySearch(results []uint32) {
	store.results = results
	store.resultIndex = -1
	store.NextResult()
}

func (store *MessageStore) ApplyFilter(results []uint32) {
	store.results = results
	store.filter = true
	store.update()
	// any marking is now invalid
	// TODO: could save that probably
	store.ClearVisualMark()
}

func (store *MessageStore) ApplyClear() {
	store.results = nil
	store.filter = false
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
	for i, uid := range store.uids {
		if store.results[len(store.results)-store.resultIndex-1] == uid {
			store.Select(len(store.uids) - i - 1)
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

func (store *MessageStore) ModifyLabels(uids []uint32, add, remove []string,
	cb func(msg types.WorkerMessage)) {
	store.worker.PostAction(&types.ModifyLabels{
		Uids:   uids,
		Add:    add,
		Remove: remove,
	}, cb)
}

func (store *MessageStore) Sort(criteria []*types.SortCriterion, cb func()) {
	store.worker.PostAction(&types.FetchDirectoryContents{
		SortCriteria: criteria,
	}, func(msg types.WorkerMessage) {
		cb()
	})
}

// returns the index of needle in haystack or -1 if not found
func (store *MessageStore) visualStartIdx() int {
	for idx, u := range store.Uids() {
		if u == store.visualStartUid {
			return idx
		}
	}
	return -1
}
