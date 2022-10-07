package widgets

import (
	"math"
	"strings"
	"sync"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib/ui"
	"git.sr.ht/~rjarry/aerc/logging"
	"github.com/gdamore/tcell/v2"
)

type ListBox struct {
	Scrollable
	title       string
	lines       []string
	selected    string
	cursorPos   int
	jump        int
	showCursor  bool
	showFilter  bool
	filterMutex sync.Mutex
	filter      *ui.TextInput
	uiConfig    *config.UIConfig
	cb          func(string)
}

func NewListBox(title string, lines []string, uiConfig *config.UIConfig, cb func(string)) *ListBox {
	lb := &ListBox{
		title:     title,
		lines:     lines,
		cursorPos: -1,
		jump:      -1,
		uiConfig:  uiConfig,
		cb:        cb,
		filter:    ui.NewTextInput("", uiConfig),
	}
	lb.filter.OnChange(func(ti *ui.TextInput) {
		var show bool
		if ti.String() == "" {
			show = false
		} else {
			show = true
		}
		lb.setShowFilterField(show)
		lb.filter.Focus(show)
		lb.Invalidate()
	})
	lb.dedup()
	return lb
}

func (lb *ListBox) dedup() {
	dedupped := make([]string, 0, len(lb.lines))
	dedup := make(map[string]struct{})
	for _, line := range lb.lines {
		if _, dup := dedup[line]; dup {
			logging.Warnf("ignore duplicate: %s", line)
			continue
		}
		dedup[line] = struct{}{}
		dedupped = append(dedupped, line)
	}
	lb.lines = dedupped
}

func (lb *ListBox) setShowFilterField(b bool) {
	lb.filterMutex.Lock()
	defer lb.filterMutex.Unlock()
	lb.showFilter = b
}

func (lb *ListBox) showFilterField() bool {
	lb.filterMutex.Lock()
	defer lb.filterMutex.Unlock()
	return lb.showFilter
}

func (lb *ListBox) Draw(ctx *ui.Context) {
	defaultStyle := lb.uiConfig.GetStyle(config.STYLE_DEFAULT)
	titleStyle := lb.uiConfig.GetStyle(config.STYLE_TITLE)
	w, h := ctx.Width(), ctx.Height()
	ctx.Fill(0, 0, w, h, ' ', defaultStyle)
	ctx.Fill(0, 0, w, 1, ' ', titleStyle)
	ctx.Printf(0, 0, titleStyle, "%s", lb.title)

	y := 0
	if lb.showFilterField() {
		y = 1
		x := ctx.Printf(0, y, defaultStyle, "Filter: ")
		lb.filter.Draw(ctx.Subcontext(x, y, w-x, 1))
	}

	lb.drawBox(ctx.Subcontext(0, y+1, w, h-(y+1)))
}

func (lb *ListBox) moveCursor(delta int) {
	list := lb.filtered()
	if len(list) == 0 {
		return
	}
	lb.cursorPos += delta
	if lb.cursorPos < 0 {
		lb.cursorPos = 0
	}
	if lb.cursorPos >= len(list) {
		lb.cursorPos = len(list) - 1
	}
	lb.selected = list[lb.cursorPos]
	lb.showCursor = true
}

func (lb *ListBox) filtered() []string {
	list := []string{}
	filterTerm := lb.filter.String()
	for _, line := range lb.lines {
		if strings.Contains(line, filterTerm) {
			list = append(list, line)
		}
	}
	return list
}

func (lb *ListBox) drawBox(ctx *ui.Context) {
	defaultStyle := lb.uiConfig.GetStyle(config.STYLE_DEFAULT)
	selectedStyle := lb.uiConfig.GetComposedStyleSelected(config.STYLE_MSGLIST_DEFAULT, nil)

	w, h := ctx.Width(), ctx.Height()
	lb.jump = h
	list := lb.filtered()

	lb.UpdateScroller(ctx.Height(), len(list))
	scroll := 0
	lb.cursorPos = -1
	for i := 0; i < len(list); i++ {
		if lb.selected == list[i] {
			scroll = i
			lb.cursorPos = i
			break
		}
	}
	lb.EnsureScroll(scroll)

	needScrollbar := lb.NeedScrollbar()
	if needScrollbar {
		w -= 1
		if w < 0 {
			w = 0
		}
	}

	if lb.lines == nil || len(list) == 0 {
		return
	}

	y := 0
	for i := lb.Scroll(); i < len(list) && y < h; i++ {
		style := defaultStyle
		if lb.selected == list[i] && lb.showCursor {
			style = selectedStyle
		}
		ctx.Printf(1, y, style, list[i])
		y += 1
	}

	if needScrollbar {
		scrollBarCtx := ctx.Subcontext(w, 0, 1, ctx.Height())
		lb.drawScrollbar(scrollBarCtx)
	}
}

func (lb *ListBox) drawScrollbar(ctx *ui.Context) {
	gutterStyle := tcell.StyleDefault
	pillStyle := tcell.StyleDefault.Reverse(true)

	// gutter
	h := ctx.Height()
	ctx.Fill(0, 0, 1, h, ' ', gutterStyle)

	// pill
	pillSize := int(math.Ceil(float64(h) * lb.PercentVisible()))
	pillOffset := int(math.Floor(float64(h) * lb.PercentScrolled()))
	ctx.Fill(0, pillOffset, 1, pillSize, ' ', pillStyle)
}

func (lb *ListBox) Invalidate() {
	ui.Invalidate()
}

func (lb *ListBox) Event(event tcell.Event) bool {
	if event, ok := event.(*tcell.EventKey); ok {
		switch event.Key() {
		case tcell.KeyCtrlP, tcell.KeyUp:
			lb.moveCursor(-1)
			lb.Invalidate()
			return true
		case tcell.KeyCtrlN, tcell.KeyDown:
			lb.moveCursor(+1)
			lb.Invalidate()
			return true
		case tcell.KeyPgUp:
			if lb.jump >= 0 {
				lb.moveCursor(-lb.jump)
				lb.Invalidate()
			}
			return true
		case tcell.KeyPgDn:
			if lb.jump >= 0 {
				lb.moveCursor(+lb.jump)
				lb.Invalidate()
			}
			return true
		case tcell.KeyEnter:
			return lb.quit(lb.selected)
		case tcell.KeyEsc:
			return lb.quit("")
		}
	}
	if lb.filter != nil {
		handled := lb.filter.Event(event)
		lb.Invalidate()
		return handled
	}
	return false
}

func (lb *ListBox) quit(s string) bool {
	lb.filter.Focus(false)
	if lb.cb != nil {
		lb.cb(s)
	}
	return true
}

func (lb *ListBox) Focus(f bool) {
	lb.filter.Focus(f)
}
