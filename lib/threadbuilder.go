package lib

import (
	"log"
	"time"

	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
	"github.com/gatherstars-com/jwz"
)

type UidStorer interface {
	Uids() []uint32
}

type ThreadBuilder struct {
	threadBlocks   map[uint32]jwz.Threadable
	messageidToUid map[string]uint32
	seen           map[uint32]bool
	store          UidStorer
	logger         *log.Logger
}

func NewThreadBuilder(store UidStorer, logger *log.Logger) *ThreadBuilder {
	tb := &ThreadBuilder{
		threadBlocks:   make(map[uint32]jwz.Threadable),
		messageidToUid: make(map[string]uint32),
		seen:           make(map[uint32]bool),
		store:          store,
		logger:         logger,
	}
	return tb
}

func (builder *ThreadBuilder) Update(msg *models.MessageInfo) {
	if msg != nil {
		if threadable := newThreadable(msg); threadable != nil {
			builder.messageidToUid[threadable.MessageThreadID()] = msg.Uid
			builder.threadBlocks[msg.Uid] = threadable
		}
	}
}

func (builder *ThreadBuilder) Threads() []*types.Thread {
	start := time.Now()

	threads := builder.buildAercThreads(builder.generateStructure())

	elapsed := time.Since(start)
	builder.logger.Println("ThreadBuilder:", len(threads), "threads created in", elapsed)

	return threads
}

func (builder *ThreadBuilder) generateStructure() jwz.Threadable {
	jwzThreads := make([]jwz.Threadable, 0, len(builder.threadBlocks))
	for _, uid := range builder.store.Uids() {
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

func (builder *ThreadBuilder) buildAercThreads(structure jwz.Threadable) []*types.Thread {
	threads := make([]*types.Thread, 0, len(builder.threadBlocks))
	if structure == nil {
		for _, uid := range builder.store.Uids() {
			threads = append(threads, &types.Thread{Uid: uid})
		}
	} else {
		// fill threads with nil messages
		for _, uid := range builder.store.Uids() {
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

	if t.IsDummy() || t.MsgInfo == nil || t.MsgInfo.Envelope == nil {
		return ""
	}
	return t.MsgInfo.Envelope.Subject
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
