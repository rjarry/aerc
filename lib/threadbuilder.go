package lib

import (
	"sync"
	"time"

	"git.sr.ht/~rjarry/aerc/lib/iterator"
	"git.sr.ht/~rjarry/aerc/logging"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
	"github.com/gatherstars-com/jwz"
)

type ThreadBuilder struct {
	sync.Mutex
	threadBlocks map[uint32]jwz.Threadable
	threadedUids []uint32
	iterFactory  iterator.Factory
}

func NewThreadBuilder(i iterator.Factory) *ThreadBuilder {
	tb := &ThreadBuilder{
		threadBlocks: make(map[uint32]jwz.Threadable),
		iterFactory:  i,
	}
	return tb
}

// Uids returns the uids in threading order
func (builder *ThreadBuilder) Uids() []uint32 {
	builder.Lock()
	defer builder.Unlock()

	if builder.threadedUids == nil {
		return []uint32{}
	}
	return builder.threadedUids
}

// Update updates the thread builder with a new message header
func (builder *ThreadBuilder) Update(msg *models.MessageInfo) {
	builder.Lock()
	defer builder.Unlock()

	if msg != nil {
		if threadable := newThreadable(msg); threadable != nil {
			builder.threadBlocks[msg.Uid] = threadable
		}
	}
}

// Threads returns a slice of threads for the given list of uids
func (builder *ThreadBuilder) Threads(uids []uint32, inverse bool, sort bool,
) []*types.Thread {
	builder.Lock()
	defer builder.Unlock()

	start := time.Now()

	threads := builder.buildAercThreads(builder.generateStructure(uids),
		uids, sort)

	// sort threads according to uid ordering
	builder.sortThreads(threads, uids)

	// rebuild uids from threads
	builder.RebuildUids(threads, inverse)

	elapsed := time.Since(start)
	logging.Infof("%d threads from %d uids created in %s", len(threads),
		len(uids), elapsed)

	return threads
}

func (builder *ThreadBuilder) generateStructure(uids []uint32) jwz.Threadable {
	jwzThreads := make([]jwz.Threadable, 0, len(builder.threadBlocks))
	for _, uid := range uids {
		if thr, ok := builder.threadBlocks[uid]; ok {
			jwzThreads = append(jwzThreads, thr)
		}
	}

	threader := jwz.NewThreader()
	threadStructure, err := threader.ThreadSlice(jwzThreads)
	if err != nil {
		logging.Errorf("failed slicing threads: %v", err)
	}
	return threadStructure
}

func (builder *ThreadBuilder) buildAercThreads(structure jwz.Threadable,
	uids []uint32, sort bool,
) []*types.Thread {
	threads := make([]*types.Thread, 0, len(builder.threadBlocks))
	if structure == nil {
		for _, uid := range uids {
			threads = append(threads, &types.Thread{Uid: uid})
		}
	} else {

		// prepare bigger function
		var bigger func(l, r *types.Thread) bool
		if sort {
			sortMap := make(map[uint32]int)
			for i, uid := range uids {
				sortMap[uid] = i
			}
			bigger = func(left, right *types.Thread) bool {
				if left == nil || right == nil {
					return false
				}
				return sortMap[left.Uid] > sortMap[right.Uid]
			}
		} else {
			bigger = func(left, right *types.Thread) bool {
				if left == nil || right == nil {
					return false
				}
				return left.Uid > right.Uid
			}
		}

		// add uids for the unfetched messages
		for _, uid := range uids {
			if _, ok := builder.threadBlocks[uid]; !ok {
				threads = append(threads, &types.Thread{Uid: uid})
			}
		}

		// build thread tree
		root := &types.Thread{Uid: 0}
		builder.buildTree(structure, root, bigger)

		// copy top-level threads to thread slice
		for thread := root.FirstChild; thread != nil; thread = thread.NextSibling {
			thread.Parent = nil
			threads = append(threads, thread)
		}

	}
	return threads
}

// buildTree recursively translates the jwz threads structure into aerc threads
func (builder *ThreadBuilder) buildTree(c jwz.Threadable, parent *types.Thread,
	bigger func(l, r *types.Thread) bool,
) {
	if c == nil || parent == nil {
		return
	}
	for node := c; node != nil; node = node.GetNext() {
		thread := parent
		if !node.IsDummy() {
			thread = builder.newThread(node, parent)
			parent.InsertCmp(thread, bigger)
		}
		builder.buildTree(node.GetChild(), thread, bigger)
	}
}

func (builder *ThreadBuilder) newThread(c jwz.Threadable, parent *types.Thread) *types.Thread {
	if threadable, ok := c.(*threadable); ok {
		return &types.Thread{Uid: threadable.MsgInfo.Uid, Parent: parent}
	}
	return nil
}

func (builder *ThreadBuilder) sortThreads(threads []*types.Thread, orderedUids []uint32) {
	types.SortThreadsBy(threads, orderedUids)
}

// RebuildUids rebuilds the uids from the given slice of threads
func (builder *ThreadBuilder) RebuildUids(threads []*types.Thread, inverse bool) {
	uids := make([]uint32, 0, len(threads))
	iterT := builder.iterFactory.NewIterator(threads)
	for iterT.Next() {
		var threaduids []uint32
		_ = iterT.Value().(*types.Thread).Walk(
			func(t *types.Thread, level int, currentErr error) error {
				threaduids = append(threaduids, t.Uid)
				return nil
			})
		if inverse {
			for j := len(threaduids) - 1; j >= 0; j-- {
				uids = append(uids, threaduids[j])
			}
		} else {
			uids = append(uids, threaduids...)
		}
	}
	result := make([]uint32, 0, len(uids))
	iterU := builder.iterFactory.NewIterator(uids)
	for iterU.Next() {
		result = append(result, iterU.Value().(uint32))
	}
	builder.threadedUids = result
}

// threadable implements the jwz.threadable interface which is required for the
// jwz threading algorithm
type threadable struct {
	MsgInfo   *models.MessageInfo
	MessageId string
	Next      jwz.Threadable
	Parent    jwz.Threadable
	Child     jwz.Threadable
	Dummy     bool
}

func newThreadable(msg *models.MessageInfo) *threadable {
	msgid, err := msg.MsgId()
	if err != nil {
		return nil
	}
	return &threadable{
		MessageId: msgid,
		MsgInfo:   msg,
		Next:      nil,
		Parent:    nil,
		Child:     nil,
		Dummy:     false,
	}
}

func (t *threadable) MessageThreadID() string {
	return t.MessageId
}

func (t *threadable) MessageThreadReferences() []string {
	if t.IsDummy() || t.MsgInfo == nil {
		return nil
	}
	irp, err := t.MsgInfo.InReplyTo()
	if err != nil {
		irp = ""
	}
	refs, err := t.MsgInfo.References()
	if err != nil || len(refs) == 0 {
		if irp == "" {
			return nil
		}
		refs = []string{irp}
	}
	return cleanRefs(t.MessageThreadID(), irp, refs)
}

// cleanRefs cleans up the references headers for threading
// 1) message-id should not be part of the references
// 2) no message-id should occur twice (avoid circularities)
// 3) in-reply-to header should not be at the beginning
func cleanRefs(m, irp string, refs []string) []string {
	considered := make(map[string]interface{})
	cleanRefs := make([]string, 0, len(refs))
	for _, r := range refs {
		if _, seen := considered[r]; r != m && !seen {
			considered[r] = nil
			cleanRefs = append(cleanRefs, r)
		}
	}
	if irp != "" && len(cleanRefs) > 0 {
		if cleanRefs[0] == irp {
			cleanRefs = append(cleanRefs[1:], irp)
		}
	}
	return cleanRefs
}

func (t *threadable) Subject() string {
	// deactivate threading by subject for now
	return ""
}

func (t *threadable) SimplifiedSubject() string {
	return ""
}

func (t *threadable) SubjectIsReply() bool {
	return false
}

func (t *threadable) SetNext(next jwz.Threadable) {
	t.Next = next
}

func (t *threadable) SetChild(kid jwz.Threadable) {
	t.Child = kid
	if kid != nil {
		kid.SetParent(t)
	}
}

func (t *threadable) SetParent(parent jwz.Threadable) {
	t.Parent = parent
}

func (t *threadable) GetNext() jwz.Threadable {
	return t.Next
}

func (t *threadable) GetChild() jwz.Threadable {
	return t.Child
}

func (t *threadable) GetParent() jwz.Threadable {
	return t.Parent
}

func (t *threadable) GetDate() time.Time {
	if t.IsDummy() {
		if t.GetChild() != nil {
			return t.GetChild().GetDate()
		}
		return time.Unix(0, 0)
	}
	if t.MsgInfo == nil || t.MsgInfo.Envelope == nil {
		return time.Unix(0, 0)
	}
	return t.MsgInfo.Envelope.Date
}

func (t *threadable) MakeDummy(forID string) jwz.Threadable {
	return &threadable{
		MessageId: forID,
		Dummy:     true,
	}
}

func (t *threadable) IsDummy() bool {
	return t.Dummy
}
