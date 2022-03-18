package lib

import (
	"log"
	"time"

	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
	"github.com/gatherstars-com/jwz"
)

type ThreadBuilder struct {
	threadBlocks   map[uint32]jwz.Threadable
	messageidToUid map[string]uint32
	seen           map[uint32]bool
	threadedUids   []uint32
	logger         *log.Logger
}

func NewThreadBuilder(logger *log.Logger) *ThreadBuilder {
	tb := &ThreadBuilder{
		threadBlocks:   make(map[uint32]jwz.Threadable),
		messageidToUid: make(map[string]uint32),
		seen:           make(map[uint32]bool),
		logger:         logger,
	}
	return tb
}

// Uids returns the uids in threading order
func (builder *ThreadBuilder) Uids() []uint32 {
	if builder.threadedUids == nil {
		return []uint32{}
	}
	return builder.threadedUids
}

// Update updates the thread builder with a new message header
func (builder *ThreadBuilder) Update(msg *models.MessageInfo) {
	if msg != nil {
		if threadable := newThreadable(msg); threadable != nil {
			builder.messageidToUid[threadable.MessageThreadID()] = msg.Uid
			builder.threadBlocks[msg.Uid] = threadable
		}
	}
}

// Threads returns a slice of threads for the given list of uids
func (builder *ThreadBuilder) Threads(uids []uint32) []*types.Thread {
	start := time.Now()

	threads := builder.buildAercThreads(builder.generateStructure(uids), uids)

	// sort threads according to uid ordering
	builder.sortThreads(threads, uids)

	// rebuild uids from threads
	builder.RebuildUids(threads)

	elapsed := time.Since(start)
	builder.logger.Println("ThreadBuilder:", len(threads), "threads created in", elapsed)

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
		builder.logger.Printf("ThreadBuilder: threading operation return error: %#v", err)
	}
	return threadStructure
}

func (builder *ThreadBuilder) buildAercThreads(structure jwz.Threadable, uids []uint32) []*types.Thread {
	threads := make([]*types.Thread, 0, len(builder.threadBlocks))
	if structure == nil {
		for _, uid := range uids {
			threads = append(threads, &types.Thread{Uid: uid})
		}
	} else {
		// fill threads with nil messages
		for _, uid := range uids {
			if _, ok := builder.threadBlocks[uid]; !ok {
				threads = append(threads, &types.Thread{Uid: uid})
			}
		}
		// append the on-the-fly created aerc threads
		root := &types.Thread{Uid: 0}
		builder.seen = make(map[uint32]bool)
		builder.buildTree(structure, root)
		for iter := root.FirstChild; iter != nil; iter = iter.NextSibling {
			iter.Parent = nil
			threads = append(threads, iter)
		}
	}
	return threads
}

// buildTree recursively translates the jwz threads structure into aerc threads
// builder.seen is used to avoid potential double-counting and should be empty
// on first call of this function
func (builder *ThreadBuilder) buildTree(treeNode jwz.Threadable, target *types.Thread) {
	if treeNode == nil {
		return
	}

	// deal with child
	uid, ok := builder.messageidToUid[treeNode.MessageThreadID()]
	if _, seen := builder.seen[uid]; ok && !seen {
		builder.seen[uid] = true
		childNode := &types.Thread{Uid: uid, Parent: target}
		target.OrderedInsert(childNode)
		builder.buildTree(treeNode.GetChild(), childNode)
	} else {
		builder.buildTree(treeNode.GetChild(), target)
	}

	// deal with siblings
	for next := treeNode.GetNext(); next != nil; next = next.GetNext() {

		uid, ok := builder.messageidToUid[next.MessageThreadID()]
		if _, seen := builder.seen[uid]; ok && !seen {
			builder.seen[uid] = true
			nn := &types.Thread{Uid: uid, Parent: target}
			target.OrderedInsert(nn)
			builder.buildTree(next.GetChild(), nn)
		} else {
			builder.buildTree(next.GetChild(), target)
		}
	}
}

func (builder *ThreadBuilder) sortThreads(threads []*types.Thread, orderedUids []uint32) {
	types.SortThreadsBy(threads, orderedUids)
}

// RebuildUids rebuilds the uids from the given slice of threads
func (builder *ThreadBuilder) RebuildUids(threads []*types.Thread) {
	uids := make([]uint32, 0, len(threads))
	for i := len(threads) - 1; i >= 0; i-- {
		threads[i].Walk(func(t *types.Thread, level int, currentErr error) error {
			uids = append(uids, t.Uid)
			return nil
		})
	}
	// copy in reverse as msgList displays backwards
	for i, j := 0, len(uids)-1; i < j; i, j = i+1, j-1 {
		uids[i], uids[j] = uids[j], uids[i]
	}
	builder.threadedUids = uids
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
	refs, err := t.MsgInfo.References()
	if err != nil || len(refs) == 0 {
		inreplyto, err := t.MsgInfo.InReplyTo()
		if err != nil {
			return nil
		}
		refs = []string{inreplyto}
	}
	return refs
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
