package widgets

import (
	"log"

	"github.com/emersion/go-imap"
	"github.com/gdamore/tcell"

	"git.sr.ht/~sircmpwn/aerc2/config"
	"git.sr.ht/~sircmpwn/aerc2/lib/ui"
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

type MessageList struct {
	conf         *config.AercConfig
	logger       *log.Logger
	onInvalidate func(d ui.Drawable)
	selected     int
	spinner      *Spinner
	store        *MessageStore
}

// TODO: fish in config
func NewMessageList(logger *log.Logger) *MessageList {
	ml := &MessageList{
		logger:   logger,
		selected: 0,
		spinner:  NewSpinner(),
	}
	ml.spinner.OnInvalidate(func(_ ui.Drawable) {
		ml.Invalidate()
	})
	// TODO: stop spinner, probably
	ml.spinner.Start()
	return ml
}

func (ml *MessageList) OnInvalidate(onInvalidate func(d ui.Drawable)) {
	ml.onInvalidate = onInvalidate
}

func (ml *MessageList) Invalidate() {
	if ml.onInvalidate != nil {
		ml.onInvalidate(ml)
	}
}

func (ml *MessageList) Draw(ctx *ui.Context) {
	ctx.Fill(0, 0, ctx.Width(), ctx.Height(), ' ', tcell.StyleDefault)

	if ml.store == nil {
		ml.spinner.Draw(ctx)
		return
	}

	var (
		needsHeaders []uint32
		row          int = 0
	)

	for i := len(ml.store.Uids) - 1; i >= 0; i-- {
		uid := ml.store.Uids[i]
		msg := ml.store.Messages[uid]

		if row >= ctx.Height() {
			break
		}

		if msg == nil {
			needsHeaders = append(needsHeaders, uid)
			ml.spinner.Draw(ctx.Subcontext(0, row, ctx.Width(), 1))
			row += 1
			continue
		}

		style := tcell.StyleDefault
		if row == ml.selected {
			style = style.Background(tcell.ColorWhite).
				Foreground(tcell.ColorBlack)
		}
		ctx.Fill(0, row, ctx.Width(), 1, ' ', style)
		ctx.Printf(0, row, style, "%s", msg.Envelope.Subject)

		row += 1
	}

	if len(needsHeaders) != 0 {
		ml.store.FetchHeaders(needsHeaders)
		ml.spinner.Start()
	} else {
		ml.spinner.Stop()
	}
}

func (ml *MessageList) SetStore(store *MessageStore) {
	ml.store = store
	if store != nil {
		ml.spinner.Stop()
	} else {
		ml.spinner.Start()
	}
	ml.Invalidate()
}
