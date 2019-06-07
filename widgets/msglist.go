package widgets

import (
	"log"

	"github.com/gdamore/tcell"

	"git.sr.ht/~sircmpwn/aerc/config"
	"git.sr.ht/~sircmpwn/aerc/lib"
	"git.sr.ht/~sircmpwn/aerc/lib/ui"
	"git.sr.ht/~sircmpwn/aerc/worker/types"
)

type MessageList struct {
	ui.Invalidatable
	conf     *config.AercConfig
	logger   *log.Logger
	height   int
	scroll   int
	selected int
	nmsgs    int
	spinner  *Spinner
	store    *lib.MessageStore
}

func NewMessageList(conf *config.AercConfig, logger *log.Logger) *MessageList {
	ml := &MessageList{
		conf:     conf,
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

func (ml *MessageList) Invalidate() {
	ml.DoInvalidate(ml)
}

func (ml *MessageList) Draw(ctx *ui.Context) {
	ml.height = ctx.Height()
	ctx.Fill(0, 0, ctx.Width(), ctx.Height(), ' ', tcell.StyleDefault)

	store := ml.Store()
	if store == nil {
		ml.spinner.Draw(ctx)
		return
	}

	var (
		needsHeaders []uint32
		row          int = 0
	)

	for i := len(store.Uids) - 1 - ml.scroll; i >= 0; i-- {
		uid := store.Uids[i]
		msg := store.Messages[uid]

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
		if _, ok := store.Deleted[msg.Uid]; ok {
			style = style.Foreground(tcell.ColorGray)
		}
		ctx.Fill(0, row, ctx.Width(), 1, ' ', style)
		fmtStr, args, err := lib.ParseIndexFormat(ml.conf, i, msg)
		if err != nil {
			ctx.Printf(0, row, style, "%v", err)
		} else {
			ctx.Printf(0, row, style, fmtStr, args...)
		}

		row += 1
	}

	if len(store.Uids) == 0 {
		msg := ml.conf.Ui.EmptyMessage
		ctx.Printf((ctx.Width()/2)-(len(msg)/2), 0,
			tcell.StyleDefault, "%s", msg)
	}

	if len(needsHeaders) != 0 {
		store.FetchHeaders(needsHeaders, nil)
		ml.spinner.Start()
	} else {
		ml.spinner.Stop()
	}
}

func (ml *MessageList) Height() int {
	return ml.height
}

func (ml *MessageList) storeUpdate(store *lib.MessageStore) {
	if ml.Store() != store {
		return
	}

	if len(store.Uids) > 0 {
		// When new messages come in, advance the cursor accordingly
		// Note that this assumes new messages are appended to the top, which
		// isn't necessarily true once we implement SORT... ideally we'd look
		// for the previously selected UID.
		if len(store.Uids) > ml.nmsgs && ml.nmsgs != 0 {
			for i := 0; i < len(store.Uids)-ml.nmsgs; i++ {
				ml.Next()
			}
		}
		if len(store.Uids) < ml.nmsgs && ml.nmsgs != 0 {
			for i := 0; i < ml.nmsgs-len(store.Uids); i++ {
				ml.Prev()
			}
		}
		ml.nmsgs = len(store.Uids)
	}

	ml.Invalidate()
}

func (ml *MessageList) SetStore(store *lib.MessageStore) {
	if ml.Store() != store {
		ml.scroll = 0
		ml.selected = 0
	}
	ml.store = store
	if store != nil {
		ml.spinner.Stop()
		ml.nmsgs = len(store.Uids)
		store.OnUpdate(ml.storeUpdate)
	} else {
		ml.spinner.Start()
	}
	ml.Invalidate()
}

func (ml *MessageList) Store() *lib.MessageStore {
	return ml.store
}

func (ml *MessageList) Empty() bool {
	store := ml.Store()
	return store == nil || len(store.Uids) == 0
}

func (ml *MessageList) Selected() *types.MessageInfo {
	store := ml.Store()
	return store.Messages[store.Uids[len(store.Uids)-ml.selected-1]]
}

func (ml *MessageList) Select(index int) {
	store := ml.Store()

	ml.selected = index
	for ; ml.selected < 0; ml.selected = len(store.Uids) + ml.selected {
	}
	if ml.selected > len(store.Uids) {
		ml.selected = len(store.Uids)
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
	store := ml.Store()

	if store == nil || len(store.Uids) == 0 {
		return
	}
	ml.selected += delta
	if ml.selected < 0 {
		ml.selected = 0
	}
	if ml.selected >= len(store.Uids) {
		ml.selected = len(store.Uids) - 1
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
