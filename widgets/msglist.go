package widgets

import (
	"fmt"
	"math"
	"strings"

	sortthread "github.com/emersion/go-imap-sortthread"
	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/lib/format"
	"git.sr.ht/~rjarry/aerc/lib/iterator"
	"git.sr.ht/~rjarry/aerc/lib/ui"
	"git.sr.ht/~rjarry/aerc/log"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

type MessageList struct {
	Scrollable
	height        int
	nmsgs         int
	spinner       *Spinner
	store         *lib.MessageStore
	isInitalizing bool
	aerc          *Aerc
}

func NewMessageList(aerc *Aerc, account *AccountView) *MessageList {
	ml := &MessageList{
		spinner:       NewSpinner(account.uiConf),
		isInitalizing: true,
		aerc:          aerc,
	}
	// TODO: stop spinner, probably
	ml.spinner.Start()
	return ml
}

func (ml *MessageList) Invalidate() {
	ui.Invalidate()
}

func (ml *MessageList) Draw(ctx *ui.Context) {
	ml.height = ctx.Height()
	uiConfig := ml.aerc.SelectedAccountUiConfig()
	ctx.Fill(0, 0, ctx.Width(), ctx.Height(), ' ',
		uiConfig.GetStyle(config.STYLE_MSGLIST_DEFAULT))

	acct := ml.aerc.SelectedAccount()
	store := ml.Store()
	if store == nil || acct == nil {
		if ml.isInitalizing {
			ml.spinner.Draw(ctx)
			return
		} else {
			ml.spinner.Stop()
			ml.drawEmptyMessage(ctx)
			return
		}
	}

	ml.UpdateScroller(ml.height, len(store.Uids()))
	if store := ml.Store(); store != nil && len(store.Uids()) > 0 {
		iter := store.UidsIterator()
		for i := 0; iter.Next(); i++ {
			if store.SelectedUid() == iter.Value().(uint32) {
				ml.EnsureScroll(i)
				break
			}
		}
	}

	textWidth := ctx.Width()
	if ml.NeedScrollbar() {
		textWidth -= 1
	}
	if textWidth < 0 {
		textWidth = 0
	}

	var (
		needsHeaders []uint32
		row          int = 0
	)

	createBaseCtx := func(uid uint32, row int) format.Ctx {
		return format.Ctx{
			FromAddress: acct.acct.From,
			AccountName: acct.Name(),
			MsgInfo:     store.Messages[uid],
			MsgNum:      row,
			MsgIsMarked: store.Marker().IsMarked(uid),
		}
	}

	if store.ThreadedView() {
		var (
			lastSubject string
			prevThread  *types.Thread
			i           int = 0
		)
		factory := iterator.NewFactory(!store.ReverseThreadOrder())
	threadLoop:
		for iter := store.ThreadsIterator(); iter.Next(); {
			var cur []*types.Thread
			err := iter.Value().(*types.Thread).Walk(
				func(t *types.Thread, _ int, _ error,
				) error {
					if t.Hidden || t.Deleted {
						return nil
					}
					cur = append(cur, t)
					return nil
				})
			if err != nil {
				log.Errorf("thread walk: %v", err)
			}
			for curIter := factory.NewIterator(cur); curIter.Next(); {
				if i < ml.Scroll() {
					i++
					continue
				}
				if thread := curIter.Value().(*types.Thread); thread != nil {
					fmtCtx := createBaseCtx(thread.Uid, row)
					fmtCtx.ThreadPrefix = threadPrefix(thread,
						store.ReverseThreadOrder())
					if fmtCtx.MsgInfo != nil && fmtCtx.MsgInfo.Envelope != nil {
						baseSubject, _ := sortthread.GetBaseSubject(
							fmtCtx.MsgInfo.Envelope.Subject)
						fmtCtx.ThreadSameSubject = baseSubject == lastSubject &&
							sameParent(thread, prevThread) &&
							!isParent(thread)
						lastSubject = baseSubject
						prevThread = thread
					}
					if ml.drawRow(textWidth, ctx, thread.Uid, row, &needsHeaders, fmtCtx) {
						break threadLoop
					}
					row += 1
				}
			}
		}
	} else {
		iter := store.UidsIterator()
		for i := 0; iter.Next(); i++ {
			if i < ml.Scroll() {
				continue
			}
			uid := iter.Value().(uint32)
			fmtCtx := createBaseCtx(uid, row)
			if ml.drawRow(textWidth, ctx, uid, row, &needsHeaders, fmtCtx) {
				break
			}
			row += 1
		}
	}

	if ml.NeedScrollbar() {
		scrollbarCtx := ctx.Subcontext(ctx.Width()-1, 0, 1, ctx.Height())
		ml.drawScrollbar(scrollbarCtx)
	}

	if len(store.Uids()) == 0 {
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

func (ml *MessageList) drawRow(textWidth int, ctx *ui.Context, uid uint32, row int, needsHeaders *[]uint32, fmtCtx format.Ctx) bool {
	store := ml.store
	msg := store.Messages[uid]
	acct := ml.aerc.SelectedAccount()

	if row >= ctx.Height() || acct == nil {
		return true
	}

	if msg == nil {
		*needsHeaders = append(*needsHeaders, uid)
		ml.spinner.Draw(ctx.Subcontext(0, row, textWidth, 1))
		return false
	}

	// TODO deprecate subject contextual UIs? Only related setting is styleset,
	// should implement a better per-message styling method
	// Check if we have any applicable ContextualUIConfigs
	uiConfig := acct.Directories().UiConfig(store.DirInfo.Name)
	if msg.Envelope != nil {
		uiConfig = uiConfig.ForSubject(msg.Envelope.Subject)
	}

	msg_styles := []config.StyleObject{}
	// unread message
	seen := false
	flagged := false
	for _, flag := range msg.Flags {
		switch flag {
		case models.SeenFlag:
			seen = true
		case models.FlaggedFlag:
			flagged = true
		}
	}

	if seen {
		msg_styles = append(msg_styles, config.STYLE_MSGLIST_READ)
	} else {
		msg_styles = append(msg_styles, config.STYLE_MSGLIST_UNREAD)
	}

	if flagged {
		msg_styles = append(msg_styles, config.STYLE_MSGLIST_FLAGGED)
	}

	// deleted message
	if _, ok := store.Deleted[msg.Uid]; ok {
		msg_styles = append(msg_styles, config.STYLE_MSGLIST_DELETED)
	}

	// marked message
	if store.Marker().IsMarked(msg.Uid) {
		msg_styles = append(msg_styles, config.STYLE_MSGLIST_MARKED)
	}

	// search result
	if store.IsResult(msg.Uid) {
		msg_styles = append(msg_styles, config.STYLE_MSGLIST_RESULT)
	}

	var style tcell.Style
	// current row
	if msg.Uid == ml.store.SelectedUid() {
		style = uiConfig.GetComposedStyleSelected(config.STYLE_MSGLIST_DEFAULT, msg_styles)
	} else {
		style = uiConfig.GetComposedStyle(config.STYLE_MSGLIST_DEFAULT, msg_styles)
	}

	ctx.Fill(0, row, ctx.Width(), 1, ' ', style)
	fmtStr, args, err := format.ParseMessageFormat(
		uiConfig.IndexFormat, uiConfig.TimestampFormat,
		uiConfig.ThisDayTimeFormat,
		uiConfig.ThisWeekTimeFormat,
		uiConfig.ThisYearTimeFormat,
		uiConfig.IconAttachment,
		fmtCtx)
	if err != nil {
		ctx.Printf(0, row, style, "%v", err)
	} else {
		line := fmt.Sprintf(fmtStr, args...)
		line = runewidth.Truncate(line, textWidth, "…")
		ctx.Printf(0, row, style, "%s", line)
	}

	return false
}

func (ml *MessageList) drawScrollbar(ctx *ui.Context) {
	gutterStyle := tcell.StyleDefault
	pillStyle := tcell.StyleDefault.Reverse(true)

	// gutter
	ctx.Fill(0, 0, 1, ctx.Height(), ' ', gutterStyle)

	// pill
	pillSize := int(math.Ceil(float64(ctx.Height()) * ml.PercentVisible()))
	pillOffset := int(math.Floor(float64(ctx.Height()) * ml.PercentScrolled()))
	ctx.Fill(0, pillOffset, 1, pillSize, ' ', pillStyle)
}

func (ml *MessageList) MouseEvent(localX int, localY int, event tcell.Event) {
	if event, ok := event.(*tcell.EventMouse); ok {
		switch event.Buttons() {
		case tcell.Button1:
			if ml.aerc == nil {
				return
			}
			selectedMsg, ok := ml.Clicked(localX, localY)
			if ok {
				ml.Select(selectedMsg)
				acct := ml.aerc.SelectedAccount()
				if acct == nil || acct.Messages().Empty() {
					return
				}
				store := acct.Messages().Store()
				msg := acct.Messages().Selected()
				if msg == nil {
					return
				}
				lib.NewMessageStoreView(msg, acct.UiConfig().AutoMarkRead,
					store, ml.aerc.Crypto, ml.aerc.DecryptKeys,
					func(view lib.MessageView, err error) {
						if err != nil {
							ml.aerc.PushError(err.Error())
							return
						}
						viewer := NewMessageViewer(acct, view)
						ml.aerc.NewTab(viewer, msg.Envelope.Subject)
					})
			}
		case tcell.WheelDown:
			if ml.store != nil {
				ml.store.Next()
			}
			ml.Invalidate()
		case tcell.WheelUp:
			if ml.store != nil {
				ml.store.Prev()
			}
			ml.Invalidate()
		}
	}
}

func (ml *MessageList) Clicked(x, y int) (int, bool) {
	store := ml.Store()
	if store == nil || ml.nmsgs == 0 || y >= ml.nmsgs {
		return 0, false
	}
	return y + ml.Scroll(), true
}

func (ml *MessageList) Height() int {
	return ml.height
}

func (ml *MessageList) storeUpdate(store *lib.MessageStore) {
	if ml.Store() != store {
		return
	}
	ml.Invalidate()
}

func (ml *MessageList) SetStore(store *lib.MessageStore) {
	if ml.Store() != store {
		ml.Scrollable = Scrollable{}
	}
	ml.store = store
	if store != nil {
		ml.spinner.Stop()
		uids := store.Uids()
		ml.nmsgs = len(uids)
		store.OnUpdate(ml.storeUpdate)
		store.OnFilterChange(func(store *lib.MessageStore) {
			if ml.Store() != store {
				return
			}
			ml.nmsgs = len(store.Uids())
		})
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
	return ml.Store().Selected()
}

func (ml *MessageList) Select(index int) {
	// Note that the msgstore.Select function expects a uid as argument
	// whereas the msglist.Select expects the message number
	store := ml.Store()
	uids := store.Uids()
	if len(uids) == 0 {
		store.Select(lib.MagicUid)
		return
	}

	iter := store.UidsIterator()

	var uid uint32
	if index < 0 {
		uid = uids[iter.EndIndex()]
	} else {
		uid = uids[iter.StartIndex()]
		for i := 0; iter.Next(); i++ {
			if i >= index {
				uid = iter.Value().(uint32)
				break
			}
		}
	}
	store.Select(uid)

	ml.Invalidate()
}

func (ml *MessageList) drawEmptyMessage(ctx *ui.Context) {
	uiConfig := ml.aerc.SelectedAccountUiConfig()
	msg := uiConfig.EmptyMessage
	ctx.Printf((ctx.Width()/2)-(len(msg)/2), 0,
		uiConfig.GetStyle(config.STYLE_MSGLIST_DEFAULT), "%s", msg)
}

func threadPrefix(t *types.Thread, reverse bool) string {
	var arrow string
	if t.Parent != nil {
		if t.NextSibling != nil {
			arrow = "├─>"
		} else {
			if reverse {
				arrow = "┌─>"
			} else {
				arrow = "└─>"
			}
		}
	}
	var prefix []string
	for n := t; n.Parent != nil; n = n.Parent {
		if n.Parent.NextSibling != nil {
			prefix = append(prefix, "│  ")
		} else {
			prefix = append(prefix, "   ")
		}
	}
	// prefix is now in a reverse order (inside --> outside), so turn it
	for i, j := 0, len(prefix)-1; i < j; i, j = i+1, j-1 {
		prefix[i], prefix[j] = prefix[j], prefix[i]
	}

	// we don't want to indent the first child, hence we strip that level
	if len(prefix) > 0 {
		prefix = prefix[1:]
	}
	ps := strings.Join(prefix, "")
	return fmt.Sprintf("%v%v", ps, arrow)
}

func sameParent(left, right *types.Thread) bool {
	return left.Root() == right.Root()
}

func isParent(t *types.Thread) bool {
	return t == t.Root()
}
