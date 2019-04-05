package widgets

import (
	"log"

	"github.com/gdamore/tcell"

	"git.sr.ht/~sircmpwn/aerc2/config"
	"git.sr.ht/~sircmpwn/aerc2/lib"
	"git.sr.ht/~sircmpwn/aerc2/lib/ui"
	"git.sr.ht/~sircmpwn/aerc2/worker/types"
)

type MessageList struct {
	conf         *config.AercConfig
	logger       *log.Logger
	height       int
	onInvalidate func(d ui.Drawable)
	scroll       int
	selected     int
	spinner      *Spinner
	store        *lib.MessageStore
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
	ml.height = ctx.Height()
	ctx.Fill(0, 0, ctx.Width(), ctx.Height(), ' ', tcell.StyleDefault)

	if ml.store == nil {
		ml.spinner.Draw(ctx)
		return
	}

	var (
		needsHeaders []uint32
		row          int = 0
	)

	for i := len(ml.store.Uids) - 1 - ml.scroll; i >= 0; i-- {
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
		if row == ml.selected-ml.scroll {
			style = style.Reverse(true)
		}
		if _, ok := ml.store.Deleted[msg.Uid]; ok {
			style = style.Foreground(tcell.ColorGray)
		}
		ctx.Fill(0, row, ctx.Width(), 1, ' ', style)
		ctx.Printf(0, row, style, "%s", msg.Envelope.Subject)

		row += 1
	}

	if len(ml.store.Uids) == 0 {
		msg := "(no messages)"
		ctx.Printf((ctx.Width()/2)-(len(msg)/2), 0,
			tcell.StyleDefault, "%s", msg)
	}

	if len(needsHeaders) != 0 {
		ml.store.FetchHeaders(needsHeaders, nil)
		ml.spinner.Start()
	} else {
		ml.spinner.Stop()
	}
}

func (ml *MessageList) Height() int {
	return ml.height
}

func (ml *MessageList) storeUpdate(store *lib.MessageStore) {
	if ml.store != store {
		return
	}
	if len(ml.store.Uids) > 0 {
		for ml.selected >= len(ml.store.Uids) {
			ml.Prev()
		}
	}
	ml.Invalidate()
}

func (ml *MessageList) SetStore(store *lib.MessageStore) {
	if ml.store == store {
		ml.scroll = 0
		ml.selected = 0
	}
	ml.store = store
	if store != nil {
		ml.spinner.Stop()
		ml.store.OnUpdate(ml.storeUpdate)
	} else {
		ml.spinner.Start()
	}
	ml.Invalidate()
}

func (ml *MessageList) Store() *lib.MessageStore {
	return ml.store
}

func (ml *MessageList) Selected() *types.MessageInfo {
	return ml.store.Messages[ml.store.Uids[len(ml.store.Uids)-ml.selected-1]]
}

func (ml *MessageList) Select(index int) {
	ml.selected = index
	for ; ml.selected < 0; ml.selected = len(ml.store.Uids) + ml.selected {
	}
	if ml.selected > len(ml.store.Uids) {
		ml.selected = len(ml.store.Uids)
	}
	// I'm too lazy to do the math right now
	for ml.selected-ml.scroll >= ml.Height() {
		ml.scroll += 1
	}
	for ml.selected-ml.scroll < 0 {
		ml.scroll -= 1
	}
}

func (ml *MessageList) nextPrev(delta int) {
	if ml.store == nil || len(ml.store.Uids) == 0 {
		return
	}
	ml.selected += delta
	if ml.selected < 0 {
		ml.selected = 0
	}
	if ml.selected >= len(ml.store.Uids) {
		ml.selected = len(ml.store.Uids) - 1
	}
	if ml.Height() != 0 {
		if ml.selected-ml.scroll >= ml.Height() {
			ml.scroll += 1
		} else if ml.selected-ml.scroll < 0 {
			ml.scroll -= 1
		}
	}
	ml.Invalidate()
}

func (ml *MessageList) Next() {
	ml.nextPrev(1)
}

func (ml *MessageList) Prev() {
	ml.nextPrev(-1)
}
