package widgets

import (
	"log"

	"github.com/gdamore/tcell"

	"git.sr.ht/~sircmpwn/aerc2/config"
	"git.sr.ht/~sircmpwn/aerc2/lib/ui"
	"git.sr.ht/~sircmpwn/aerc2/worker/types"
)

type MessageStore struct {
	DirInfo  types.DirectoryInfo
	Messages map[uint64]*types.MessageInfo
}

func NewMessageStore(dirInfo *types.DirectoryInfo) *MessageStore {
	return &MessageStore{DirInfo: *dirInfo}
}

func (store *MessageStore) Update(msg types.WorkerMessage) {
	switch msg := msg.(type) {
	case *types.DirectoryInfo:
		store.DirInfo = *msg
		break
	case *types.DirectoryContents:
		newMap := make(map[uint64]*types.MessageInfo)
		for _, uid := range msg.Uids {
			if msg, ok := store.Messages[uid]; ok {
				newMap[uid] = msg
			} else {
				newMap[uid] = nil
			}
		}
		store.Messages = newMap
		break
	case *types.MessageInfo:
		store.Messages[msg.Uid] = msg
		break
	}
}

type MessageList struct {
	conf         *config.AercConfig
	logger       *log.Logger
	onInvalidate func(d ui.Drawable)
	spinner      *Spinner
	store        *MessageStore
	worker       *types.Worker
}

// TODO: fish in config
func NewMessageList(logger *log.Logger, worker *types.Worker) *MessageList {
	ml := &MessageList{
		logger:  logger,
		spinner: NewSpinner(),
		worker:  worker,
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
		needsHeaders []uint64
		row          int = 0
	)

	for uid, msg := range ml.store.Messages {
		if row >= ctx.Height() {
			break
		}

		if msg == nil {
			needsHeaders = append(needsHeaders, uid)
			ml.spinner.Draw(ctx.Subcontext(0, row, ctx.Width(), 1))
		}

		row += 1
	}

	if len(needsHeaders) != 0 {
		ml.spinner.Start()
	} else {
		ml.spinner.Stop()
	}

	// TODO: Fetch these messages
}

func (ml *MessageList) SetStore(store *MessageStore) {
	if ml.store == store {
		return
	}

	ml.store = store
	if store != nil {
		ml.spinner.Stop()
	} else {
		ml.spinner.Start()
	}
	ml.Invalidate()
}
