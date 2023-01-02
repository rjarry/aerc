package lib

import (
	"io"
	"sync"
	"time"

	"git.sr.ht/~rjarry/aerc/lib/iterator"
	"git.sr.ht/~rjarry/aerc/lib/marker"
	"git.sr.ht/~rjarry/aerc/lib/sort"
	"git.sr.ht/~rjarry/aerc/lib/ui"
	"git.sr.ht/~rjarry/aerc/log"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

// Accesses to fields must be guarded by MessageStore.Lock/Unlock
type MessageStore struct {
	sync.Mutex
	Deleted  map[uint32]interface{}
	DirInfo  models.DirectoryInfo
	Messages map[uint32]*models.MessageInfo
	Sorting  bool

	// Ordered list of known UIDs
	uids    []uint32
	threads []*types.Thread

	selectedUid   uint32
	bodyCallbacks map[uint32][]func(*types.FullMessage)

	// marking
	marker marker.Marker

	// Search/filter results
	results     []uint32
	resultIndex int
	filter      []string

	sortCriteria []*types.SortCriterion
	sortDefault  []*types.SortCriterion

	threadedView       bool
	reverseThreadOrder bool
	sortThreadSiblings bool
	buildThreads       bool
	builder            *ThreadBuilder

	// Map of uids we've asked the worker to fetch
	onUpdate       func(store *MessageStore) // TODO: multiple onUpdate handlers
	onFilterChange func(store *MessageStore)
	onUpdateDirs   func()
	pendingBodies  map[uint32]interface{}
	pendingHeaders map[uint32]interface{}
	worker         *types.Worker

	needsFlags         []uint32
	fetchFlagsDebounce *time.Timer
	fetchFlagsDelay    time.Duration

	triggerNewEmail        func(*models.MessageInfo)
	triggerDirectoryChange func()

	threadBuilderDebounce *time.Timer
	threadBuilderDelay    time.Duration
	threadCallback        func()

	// threads mutex protects the store.threads and store.threadCallback
	threadsMutex sync.Mutex

	iterFactory iterator.Factory
	onSelect    func(*models.MessageInfo)
}

const MagicUid = 0xFFFFFFFF

func NewMessageStore(worker *types.Worker,
	dirInfo *models.DirectoryInfo,
	defaultSortCriteria []*types.SortCriterion,
	thread bool, clientThreads bool, clientThreadsDelay time.Duration,
	reverseOrder bool, reverseThreadOrder bool, sortThreadSiblings bool,
	triggerNewEmail func(*models.MessageInfo),
	triggerDirectoryChange func(), onSelect func(*models.MessageInfo),
) *MessageStore {
	if !dirInfo.Caps.Thread {
		clientThreads = true
	}

	return &MessageStore{
		Deleted:  make(map[uint32]interface{}),
		DirInfo:  *dirInfo,
		Messages: make(map[uint32]*models.MessageInfo),

		selectedUid: MagicUid,

		bodyCallbacks: make(map[uint32][]func(*types.FullMessage)),

		threadedView:       thread,
		buildThreads:       clientThreads,
		reverseThreadOrder: reverseThreadOrder,
		sortThreadSiblings: sortThreadSiblings,

		filter:       []string{"filter"},
		sortCriteria: defaultSortCriteria,
		sortDefault:  defaultSortCriteria,

		pendingBodies:  make(map[uint32]interface{}),
		pendingHeaders: make(map[uint32]interface{}),
		worker:         worker,

		needsFlags:      []uint32{},
		fetchFlagsDelay: 50 * time.Millisecond,

		triggerNewEmail:        triggerNewEmail,
		triggerDirectoryChange: triggerDirectoryChange,

		threadBuilderDelay: clientThreadsDelay,

		iterFactory: iterator.NewFactory(reverseOrder),
		onSelect:    onSelect,
	}
}

func (store *MessageStore) FetchHeaders(uids []uint32,
	cb func(types.WorkerMessage),
) {
	// TODO: this could be optimized by pre-allocating toFetch and trimming it
	// at the end. In practice we expect to get most messages back in one frame.
	var toFetch []uint32
	for _, uid := range uids {
		if _, ok := store.pendingHeaders[uid]; !ok {
			toFetch = append(toFetch, uid)
			store.pendingHeaders[uid] = nil
		}
	}
	if len(toFetch) > 0 {
		store.worker.PostAction(&types.FetchMessageHeaders{Uids: toFetch}, func(msg types.WorkerMessage) {
			if _, ok := msg.(*types.Error); ok {
				for _, uid := range toFetch {
					delete(store.pendingHeaders, uid)
				}
			}
			if cb != nil {
				cb(msg)
			}
		})
	}
}

func (store *MessageStore) FetchFull(uids []uint32, cb func(*types.FullMessage)) {
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
					store.bodyCallbacks[uid] = []func(*types.FullMessage){cb}
				}
			}
		}
	}
	if len(toFetch) > 0 {
		store.worker.PostAction(&types.FetchFullMessages{
			Uids: toFetch,
		}, func(msg types.WorkerMessage) {
			if _, ok := msg.(*types.Error); ok {
				for _, uid := range toFetch {
					delete(store.pendingBodies, uid)
					delete(store.bodyCallbacks, uid)
				}
			}
		})
	}
}

func (store *MessageStore) FetchBodyPart(uid uint32, part []int, cb func(io.Reader)) {
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
	to.Labels = from.Labels
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
	updateThreads := false
	directoryChange := false
	switch msg := msg.(type) {
	case *types.DirectoryInfo:
		store.DirInfo = *msg.Info
		if !msg.SkipSort {
			store.Sort(store.sortCriteria, nil)
		}
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
		if store.threadedView {
			store.runThreadBuilderNow()
		}
	case *types.DirectoryThreaded:
		newMap := make(map[uint32]*models.MessageInfo)

		store.builder = NewThreadBuilder(store.iterFactory)
		store.builder.RebuildUids(msg.Threads, store.reverseThreadOrder)
		store.uids = store.builder.Uids()
		store.threads = msg.Threads

		for _, uid := range store.uids {
			if msg, ok := store.Messages[uid]; ok {
				newMap[uid] = msg
			} else {
				newMap[uid] = nil
				directoryChange = true
			}
		}

		store.Messages = newMap
		update = true
	case *types.MessageInfo:
		if existing, ok := store.Messages[msg.Info.Uid]; ok && existing != nil {
			merge(existing, msg.Info)
		} else if msg.Info.Envelope != nil {
			store.Messages[msg.Info.Uid] = msg.Info
			if store.selectedUid == msg.Info.Uid {
				store.onSelect(msg.Info)
			}
		}
		if msg.NeedsFlags {
			store.Lock()
			store.needsFlags = append(store.needsFlags, msg.Info.Uid)
			store.Unlock()
			store.fetchFlags()
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
		}
		if store.builder != nil {
			store.builder.Update(msg.Info)
		}
		update = true
		updateThreads = true
	case *types.FullMessage:
		if _, ok := store.pendingBodies[msg.Content.Uid]; ok {
			delete(store.pendingBodies, msg.Content.Uid)
			if cbs, ok := store.bodyCallbacks[msg.Content.Uid]; ok {
				for _, cb := range cbs {
					cb(msg)
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
		if len(uids) == 0 {
			store.Select(MagicUid)
		}

		var newResults []uint32
		for _, res := range store.results {
			if _, deleted := toDelete[res]; !deleted {
				newResults = append(newResults, res)
			}
		}
		store.results = newResults

		for _, thread := range store.Threads() {
			_ = thread.Walk(func(t *types.Thread, _ int, _ error) error {
				if _, deleted := toDelete[t.Uid]; deleted {
					t.Deleted = true
				}
				return nil
			})
		}

		update = true
		updateThreads = true
	}

	if update {
		store.update(updateThreads)
	}

	if directoryChange && store.triggerDirectoryChange != nil {
		store.triggerDirectoryChange()
	}
}

func (store *MessageStore) OnUpdate(fn func(store *MessageStore)) {
	store.onUpdate = fn
}

func (store *MessageStore) OnFilterChange(fn func(store *MessageStore)) {
	store.onFilterChange = fn
}

func (store *MessageStore) OnUpdateDirs(fn func()) {
	store.onUpdateDirs = fn
}

func (store *MessageStore) update(threads bool) {
	if store.onUpdate != nil {
		store.onUpdate(store)
	}
	if store.onUpdateDirs != nil {
		store.onUpdateDirs()
	}
	if store.ThreadedView() && threads {
		switch {
		case store.BuildThreads():
			store.runThreadBuilder()
		default:
			if store.builder == nil {
				store.builder = NewThreadBuilder(store.iterFactory)
			}
			store.builder.RebuildUids(store.Threads(), store.reverseThreadOrder)
		}
	}
}

func (store *MessageStore) SetReverseThreadOrder(reverse bool) {
	store.reverseThreadOrder = reverse
}

func (store *MessageStore) ReverseThreadOrder() bool {
	return store.reverseThreadOrder
}

func (store *MessageStore) SetThreadedView(thread bool) {
	store.threadedView = thread
	if store.buildThreads {
		if store.threadedView {
			store.runThreadBuilder()
		} else if store.threadBuilderDebounce != nil {
			store.threadBuilderDebounce.Stop()
		}
		return
	}
	store.Sort(store.sortCriteria, nil)
}

func (store *MessageStore) Threads() []*types.Thread {
	store.threadsMutex.Lock()
	defer store.threadsMutex.Unlock()
	return store.threads
}

func (store *MessageStore) ThreadsIterator() iterator.Iterator {
	store.threadsMutex.Lock()
	defer store.threadsMutex.Unlock()
	return store.iterFactory.NewIterator(store.threads)
}

func (store *MessageStore) ThreadedView() bool {
	return store.threadedView
}

func (store *MessageStore) BuildThreads() bool {
	return store.buildThreads
}

func (store *MessageStore) runThreadBuilder() {
	if store.builder == nil {
		store.builder = NewThreadBuilder(store.iterFactory)
		for _, msg := range store.Messages {
			store.builder.Update(msg)
		}
	}
	if store.threadBuilderDebounce != nil {
		if store.threadBuilderDebounce.Stop() {
			log.Tracef("thread builder debounced")
		}
	}
	store.threadBuilderDebounce = time.AfterFunc(store.threadBuilderDelay, func() {
		store.runThreadBuilderNow()
		ui.QueueRedraw()
	})
}

// runThreadBuilderNow runs the threadbuilder without any debounce logic
func (store *MessageStore) runThreadBuilderNow() {
	if store.builder == nil {
		store.builder = NewThreadBuilder(store.iterFactory)
		for _, msg := range store.Messages {
			store.builder.Update(msg)
		}
	}
	// build new threads
	th := store.builder.Threads(store.uids, store.reverseThreadOrder,
		store.sortThreadSiblings)

	// save local threads to the message store variable and
	// run callback if defined (callback should reposition cursor)
	store.threadsMutex.Lock()
	store.threads = th
	if store.threadCallback != nil {
		store.threadCallback()
	}
	store.threadsMutex.Unlock()

	// invalidate message list
	if store.onUpdate != nil {
		store.onUpdate(store)
	}
}

// SelectedThread returns the thread with the UID from the selected message
func (store *MessageStore) SelectedThread() *types.Thread {
	var thread *types.Thread
	for _, root := range store.Threads() {
		found := false
		err := root.Walk(func(t *types.Thread, _ int, _ error) error {
			if t.Uid == store.SelectedUid() {
				thread = t
				found = true
			}
			return nil
		})
		if err != nil {
			log.Errorf("SelectedThread failed: %v", err)
		}
		if found {
			break
		}
	}
	return thread
}

func (store *MessageStore) Delete(uids []uint32,
	cb func(msg types.WorkerMessage),
) {
	for _, uid := range uids {
		store.Deleted[uid] = nil
	}

	store.worker.PostAction(&types.DeleteMessages{Uids: uids},
		func(msg types.WorkerMessage) {
			if _, ok := msg.(*types.Error); ok {
				store.revertDeleted(uids)
			}
			if _, ok := msg.(*types.Unsupported); ok {
				store.revertDeleted(uids)
			}
			cb(msg)
		})
}

func (store *MessageStore) revertDeleted(uids []uint32) {
	for _, uid := range uids {
		delete(store.Deleted, uid)
	}
}

func (store *MessageStore) Copy(uids []uint32, dest string, createDest bool,
	cb func(msg types.WorkerMessage),
) {
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
	cb func(msg types.WorkerMessage),
) {
	for _, uid := range uids {
		store.Deleted[uid] = nil
	}

	if createDest {
		store.worker.PostAction(&types.CreateDirectory{
			Directory: dest,
			Quiet:     true,
		}, nil) // quiet doesn't return an error, don't want the done cb here
	}

	store.worker.PostAction(&types.MoveMessages{
		Destination: dest,
		Uids:        uids,
	}, func(msg types.WorkerMessage) {
		switch msg.(type) {
		case *types.Error:
			store.revertDeleted(uids)
			cb(msg)
		case *types.Done:
			cb(msg)
		}
	})
}

func (store *MessageStore) Flag(uids []uint32, flag models.Flag,
	enable bool, cb func(msg types.WorkerMessage),
) {
	store.worker.PostAction(&types.FlagMessages{
		Enable: enable,
		Flag:   flag,
		Uids:   uids,
	}, cb)
}

func (store *MessageStore) Answered(uids []uint32, answered bool,
	cb func(msg types.WorkerMessage),
) {
	store.worker.PostAction(&types.AnsweredMessages{
		Answered: answered,
		Uids:     uids,
	}, cb)
}

func (store *MessageStore) Uids() []uint32 {
	if store.ThreadedView() && store.builder != nil {
		if uids := store.builder.Uids(); len(uids) > 0 {
			return uids
		}
	}
	return store.uids
}

func (store *MessageStore) UidsIterator() iterator.Iterator {
	return store.iterFactory.NewIterator(store.Uids())
}

func (store *MessageStore) Selected() *models.MessageInfo {
	return store.Messages[store.selectedUid]
}

func (store *MessageStore) SelectedUid() uint32 {
	if store.selectedUid == MagicUid && len(store.Uids()) > 0 {
		iter := store.UidsIterator()
		store.Select(store.Uids()[iter.StartIndex()])
	}
	return store.selectedUid
}

func (store *MessageStore) Select(uid uint32) {
	store.threadsMutex.Lock()
	if store.threadCallback != nil {
		store.threadCallback = nil
	}
	store.threadsMutex.Unlock()
	store.selectPriv(uid)
}

func (store *MessageStore) selectPriv(uid uint32) {
	store.selectedUid = uid
	if store.marker != nil {
		store.marker.UpdateVisualMark()
	}
	if store.onSelect != nil {
		store.onSelect(store.Selected())
	}
}

func (store *MessageStore) NextPrev(delta int) {
	uids := store.Uids()
	if len(uids) == 0 {
		return
	}

	iter := store.iterFactory.NewIterator(uids)

	newIdx := store.FindIndexByUid(store.SelectedUid())
	if newIdx < 0 {
		store.Select(uids[iter.StartIndex()])
		return
	}
	newIdx = iterator.MoveIndex(
		newIdx,
		delta,
		iter,
		iterator.FixBounds,
	)
	store.Select(uids[newIdx])

	if store.BuildThreads() && store.ThreadedView() {
		store.threadsMutex.Lock()
		store.threadCallback = func() {
			if uids := store.Uids(); len(uids) > newIdx {
				store.selectPriv(uids[newIdx])
			}
		}
		store.threadsMutex.Unlock()
	}

	if store.marker != nil {
		store.marker.UpdateVisualMark()
	}
	store.updateResults()
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
		if msg, ok := msg.(*types.SearchResults); ok {
			allowedUids := store.Uids()
			uids := make([]uint32, 0, len(msg.Uids))
			for _, uid := range msg.Uids {
				for _, uidCheck := range allowedUids {
					if uid == uidCheck {
						uids = append(uids, uid)
						break
					}
				}
			}
			sort.SortBy(uids, allowedUids)
			cb(uids)
		}
	})
}

func (store *MessageStore) ApplySearch(results []uint32) {
	store.results = results
	store.resultIndex = -1
	store.NextResult()
}

// IsResult returns true if uid is a search result
func (store *MessageStore) IsResult(uid uint32) bool {
	for _, hit := range store.results {
		if hit == uid {
			return true
		}
	}
	return false
}

func (store *MessageStore) SetFilter(args []string) {
	store.filter = append(store.filter, args...)
}

func (store *MessageStore) ApplyClear() {
	store.filter = []string{"filter"}
	store.results = nil
	if store.onFilterChange != nil {
		store.onFilterChange(store)
	}
	store.Sort(store.sortDefault, nil)
}

func (store *MessageStore) updateResults() {
	if len(store.results) == 0 || store.resultIndex < 0 {
		return
	}
	uid := store.SelectedUid()
	for i, u := range store.results {
		if uid == u {
			store.resultIndex = i
			break
		}
	}
}

func (store *MessageStore) nextPrevResult(delta int) {
	if len(store.results) == 0 {
		return
	}
	iter := store.iterFactory.NewIterator(store.results)
	if store.resultIndex < 0 {
		store.resultIndex = iter.StartIndex()
	} else {
		store.resultIndex = iterator.MoveIndex(
			store.resultIndex,
			delta,
			iter,
			iterator.WrapBounds,
		)
	}
	store.Select(store.results[store.resultIndex])
	store.update(false)
}

func (store *MessageStore) NextResult() {
	store.nextPrevResult(1)
}

func (store *MessageStore) PrevResult() {
	store.nextPrevResult(-1)
}

func (store *MessageStore) ModifyLabels(uids []uint32, add, remove []string,
	cb func(msg types.WorkerMessage),
) {
	store.worker.PostAction(&types.ModifyLabels{
		Uids:   uids,
		Add:    add,
		Remove: remove,
	}, cb)
}

func (store *MessageStore) Sort(criteria []*types.SortCriterion, cb func(types.WorkerMessage)) {
	store.sortCriteria = criteria
	store.Sorting = true

	handle_return := func(msg types.WorkerMessage) {
		store.Select(store.SelectedUid())
		store.Sorting = false
		if cb != nil {
			cb(msg)
		}
	}

	if store.threadedView && !store.buildThreads {
		store.worker.PostAction(&types.FetchDirectoryThreaded{
			SortCriteria:   criteria,
			FilterCriteria: store.filter,
		}, handle_return)
	} else {
		store.worker.PostAction(&types.FetchDirectoryContents{
			SortCriteria:   criteria,
			FilterCriteria: store.filter,
		}, handle_return)
	}
}

func (store *MessageStore) GetCurrentSortCriteria() []*types.SortCriterion {
	return store.sortCriteria
}

func (store *MessageStore) SetMarker(m marker.Marker) {
	store.marker = m
}

func (store *MessageStore) Marker() marker.Marker {
	if store.marker == nil {
		store.marker = marker.New(store)
	}
	return store.marker
}

// FindIndexByUid returns the index in store.Uids() or -1 if not found
func (store *MessageStore) FindIndexByUid(uid uint32) int {
	for idx, u := range store.Uids() {
		if u == uid {
			return idx
		}
	}
	return -1
}

// Capabilities returns a models.Capabilities struct or nil if not available
func (store *MessageStore) Capabilities() *models.Capabilities {
	return store.DirInfo.Caps
}

// SelectedIndex returns the index of the selected message in the uid list or
// -1 if not found
func (store *MessageStore) SelectedIndex() int {
	return store.FindIndexByUid(store.selectedUid)
}

func (store *MessageStore) fetchFlags() {
	if store.fetchFlagsDebounce != nil {
		store.fetchFlagsDebounce.Stop()
	}
	store.fetchFlagsDebounce = time.AfterFunc(store.fetchFlagsDelay, func() {
		store.Lock()
		store.worker.PostAction(&types.FetchMessageFlags{
			Uids: store.needsFlags,
		}, nil)
		store.needsFlags = []uint32{}
		store.Unlock()
	})
}
