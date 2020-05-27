package widgets

import (
	"fmt"
	"log"

	"github.com/gdamore/tcell"
	"github.com/mattn/go-runewidth"

	"git.sr.ht/~sircmpwn/aerc/config"
	"git.sr.ht/~sircmpwn/aerc/lib"
	"git.sr.ht/~sircmpwn/aerc/lib/format"
	"git.sr.ht/~sircmpwn/aerc/lib/ui"
	"git.sr.ht/~sircmpwn/aerc/models"
)

type MessageList struct {
	ui.Invalidatable
	conf          *config.AercConfig
	logger        *log.Logger
	height        int
	scroll        int
	nmsgs         int
	spinner       *Spinner
	store         *lib.MessageStore
	isInitalizing bool
	aerc          *Aerc
}

func NewMessageList(conf *config.AercConfig, logger *log.Logger, aerc *Aerc) *MessageList {
	ml := &MessageList{
		conf:          conf,
		logger:        logger,
		spinner:       NewSpinner(&conf.Ui),
		isInitalizing: true,
		aerc:          aerc,
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
	ctx.Fill(0, 0, ctx.Width(), ctx.Height(), ' ',
		ml.aerc.SelectedAccount().UiConfig().GetStyle(config.STYLE_MSGLIST_DEFAULT))

	store := ml.Store()
	if store == nil {
		if ml.isInitalizing {
			ml.spinner.Draw(ctx)
			return
		} else {
			ml.spinner.Stop()
			ml.drawEmptyMessage(ctx)
			return
		}
	}

	var (
		needsHeaders []uint32
		row          int = 0
	)
	uids := store.Uids()

	for i := len(uids) - 1 - ml.scroll; i >= 0; i-- {
		uid := uids[i]
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

		uiConfig := ml.conf.GetUiConfig(map[config.ContextType]string{
			config.UI_CONTEXT_ACCOUNT: ml.aerc.SelectedAccount().AccountConfig().Name,
			config.UI_CONTEXT_FOLDER:  ml.aerc.SelectedAccount().Directories().Selected(),
			config.UI_CONTEXT_SUBJECT: msg.Envelope.Subject,
		})

		so := config.STYLE_MSGLIST_DEFAULT

		// deleted message
		if _, ok := store.Deleted[msg.Uid]; ok {
			so = config.STYLE_MSGLIST_DELETED
		}
		// unread message
		seen := false
		flaged := false
		for _, flag := range msg.Flags {
			switch flag {
			case models.SeenFlag:
				seen = true
			case models.FlaggedFlag:
				flaged = true
			}
		}
		if !seen {
			so = config.STYLE_MSGLIST_UNREAD
		}

		if flaged {
			so = config.STYLE_MSGLIST_FLAGGED
		}

		// marked message
		if store.IsMarked(msg.Uid) {
			so = config.STYLE_MSGLIST_MARKED
		}

		style := uiConfig.GetStyle(so)

		// current row
		if row == ml.store.SelectedIndex()-ml.scroll {
			style = uiConfig.GetStyleSelected(so)
		}

		ctx.Fill(0, row, ctx.Width(), 1, ' ', style)
		fmtStr, args, err := format.ParseMessageFormat(
			ml.aerc.SelectedAccount().acct.From,
			uiConfig.IndexFormat,
			uiConfig.TimestampFormat, "", i, msg, store.IsMarked(uid))
		if err != nil {
			ctx.Printf(0, row, style, "%v", err)
		} else {
			line := fmt.Sprintf(fmtStr, args...)
			line = runewidth.Truncate(line, ctx.Width(), "…")
			ctx.Printf(0, row, style, "%s", line)
		}

		row += 1
	}

	if len(uids) == 0 {
		if store.Sorting {
			ml.spinner.Start()
			ml.spinner.Draw(ctx)
			return
		} else {
			ml.drawEmptyMessage(ctx)
		}
	}

	if len(needsHeaders) != 0 {
		store.FetchHeaders(needsHeaders, nil)
		ml.spinner.Start()
	} else {
		ml.spinner.Stop()
	}
}

func (ml *MessageList) MouseEvent(localX int, localY int, event tcell.Event) {
	switch event := event.(type) {
	case *tcell.EventMouse:
		switch event.Buttons() {
		case tcell.Button1:
			if ml.aerc == nil {
				return
			}
			selectedMsg, ok := ml.Clicked(localX, localY)
			if ok {
				ml.Select(selectedMsg)
				acct := ml.aerc.SelectedAccount()
				if acct.Messages().Empty() {
					return
				}
				store := acct.Messages().Store()
				msg := acct.Messages().Selected()
				if msg == nil {
					return
				}
				lib.NewMessageStoreView(msg, store, ml.aerc.DecryptKeys,
					func(view lib.MessageView, err error) {
						if err != nil {
							ml.aerc.PushError(err.Error())
							return
						}
						viewer := NewMessageViewer(acct, ml.aerc.Config(), view)
						ml.aerc.NewTab(viewer, msg.Envelope.Subject)
					})
			}
		case tcell.WheelDown:
			if ml.store != nil {
				ml.store.Next()
			}
			ml.Scroll()
		case tcell.WheelUp:
			if ml.store != nil {
				ml.store.Prev()
			}
			ml.Scroll()
		}
	}
}

func (ml *MessageList) Clicked(x, y int) (int, bool) {
	store := ml.Store()
	if store == nil || ml.nmsgs == 0 || y >= ml.nmsgs {
		return 0, false
	}
	return y + ml.scroll, true
}

func (ml *MessageList) Height() int {
	return ml.height
}

func (ml *MessageList) storeUpdate(store *lib.MessageStore) {
	if ml.Store() != store {
		return
	}
	uids := store.Uids()

	if len(uids) > 0 {
		// When new messages come in, advance the cursor accordingly
		// Note that this assumes new messages are appended to the top, which
		// isn't necessarily true once we implement SORT... ideally we'd look
		// for the previously selected UID.
		if len(uids) > ml.nmsgs && ml.nmsgs != 0 {
			for i := 0; i < len(uids)-ml.nmsgs; i++ {
				ml.Store().Next()
			}
		}
		if len(uids) < ml.nmsgs && ml.nmsgs != 0 {
			for i := 0; i < ml.nmsgs-len(uids); i++ {
				ml.Store().Prev()
			}
		}
		ml.nmsgs = len(uids)
	}

	ml.Scroll()
	ml.Invalidate()
}

func (ml *MessageList) SetStore(store *lib.MessageStore) {
	if ml.Store() != store {
		ml.scroll = 0
	}
	ml.store = store
	if store != nil {
		ml.spinner.Stop()
		ml.nmsgs = len(store.Uids())
		store.OnUpdate(ml.storeUpdate)
	} else {
		ml.spinner.Start()
	}
	ml.Invalidate()
}

func (ml *MessageList) SetInitDone() {
	ml.isInitalizing = false
}

func (ml *MessageList) Store() *lib.MessageStore {
	return ml.store
}

func (ml *MessageList) Empty() bool {
	store := ml.Store()
	return store == nil || len(store.Uids()) == 0
}

func (ml *MessageList) Selected() *models.MessageInfo {
	store := ml.Store()
	uids := store.Uids()
	return store.Messages[uids[len(uids)-ml.store.SelectedIndex()-1]]
}

func (ml *MessageList) Select(index int) {
	store := ml.Store()
	store.Select(index)
	ml.Scroll()
}

func (ml *MessageList) Scroll() {
	store := ml.Store()

	if store == nil || len(store.Uids()) == 0 {
		return
	}
	if ml.Height() != 0 {
		// I'm too lazy to do the math right now
		for store.SelectedIndex()-ml.scroll >= ml.Height() {
			ml.scroll += 1
		}
		for store.SelectedIndex()-ml.scroll < 0 {
			ml.scroll -= 1
		}
	}
	ml.Invalidate()
}

func (ml *MessageList) drawEmptyMessage(ctx *ui.Context) {
	uiConfig := ml.aerc.SelectedAccount().UiConfig()
	msg := uiConfig.EmptyMessage
	ctx.Printf((ctx.Width()/2)-(len(msg)/2), 0,
		uiConfig.GetStyle(config.STYLE_MSGLIST_DEFAULT), "%s", msg)
}
