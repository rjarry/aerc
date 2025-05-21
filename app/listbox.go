package app

import (
	"math"
	"strings"
	"sync"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/lib/ui"
	"git.sr.ht/~rockorager/vaxis"
	"github.com/mattn/go-runewidth"
)

type ListBox struct {
	Scrollable
	title       string
	lines       []string
	selected    string
	cursorPos   int
	horizPos    int
	jump        int
	showCursor  bool
	showFilter  bool
	filterMutex sync.Mutex
	filter      *ui.TextInput
	uiConfig    *config.UIConfig
	textFilter  func([]string, string) []string
	cb          func(string)
}

func NewListBox(title string, lines []string, uiConfig *config.UIConfig, cb func(string)) *ListBox {
	lb := &ListBox{
		title:      title,
		lines:      lines,
		cursorPos:  -1,
		jump:       -1,
		uiConfig:   uiConfig,
		textFilter: nil,
		cb:         cb,
		filter:     ui.NewTextInput("", uiConfig),
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

func (lb *ListBox) SetTextFilter(fn func([]string, string) []string) *ListBox {
	lb.textFilter = fn
	return lb
}

func (lb *ListBox) dedup() {
	dedupped := make([]string, 0, len(lb.lines))
	dedup := make(map[string]struct{})
	for _, line := range lb.lines {
		if _, dup := dedup[line]; dup {
			log.Warnf("ignore duplicate: %s", line)
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
		x := ctx.Printf(0, y, defaultStyle, "Filter (%d/%d): ",
			len(lb.filtered()), len(lb.lines))
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
	lb.horizPos = 0
}

func (lb *ListBox) moveHorizontal(delta int) {
	lb.horizPos += delta
	if lb.horizPos > len(lb.selected) {
		lb.horizPos = len(lb.selected)
	}
	if lb.horizPos < 0 {
		lb.horizPos = 0
	}
}

func (lb *ListBox) filtered() []string {
	term := lb.filter.String()

	if lb.textFilter != nil {
		return lb.textFilter(lb.lines, term)
	}

	list := make([]string, 0, len(lb.lines))
	for _, line := range lb.lines {
		if strings.Contains(line, term) {
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
	if len(list) > 0 {
		if lb.cursorPos == -1 {
			// The list is not empty and we did not find the selection, if any,
			// so clear it.
			lb.selected = ""
			// Select the fist matching item to avoid the user needing to
			// explicitly select it in case it matches what they want.
			lb.moveCursor(0)
		}
	} else {
		// The list is now empty, the selection, if any, needs to be cleared.
		lb.selected = ""
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
		line := runewidth.Truncate(list[i], w-1, "❯")
		if lb.selected == list[i] && lb.showCursor {
			style = selectedStyle
			if len(list[i]) > w {
				if len(list[i])-lb.horizPos < w {
					lb.horizPos = len(list[i]) - w + 1
				}
				rest := list[i][lb.horizPos:]
				line = runewidth.Truncate(rest,
					w-1, "❯")
				if lb.horizPos > 0 && len(line) > 0 {
					line = "❮" + line[1:]
				}
			}
		}
		ctx.Printf(1, y, style, "%s", line)
		y += 1
	}

	if needScrollbar {
		scrollBarCtx := ctx.Subcontext(w, 0, 1, ctx.Height())
		lb.drawScrollbar(scrollBarCtx)
	}
}

func (lb *ListBox) drawScrollbar(ctx *ui.Context) {
	gutterStyle := vaxis.Style{}
	pillStyle := vaxis.Style{Attribute: vaxis.AttrReverse}

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

func (lb *ListBox) Event(event vaxis.Event) bool {
	showFilter := lb.showFilterField()
	if key, ok := event.(vaxis.Key); ok {
		switch {
		case key.Matches(vaxis.KeyLeft):
			if showFilter {
				break
			}
			lb.moveHorizontal(-1)
			lb.Invalidate()
			return true
		case key.Matches(vaxis.KeyRight):
			if showFilter {
				break
			}
			lb.moveHorizontal(+1)
			lb.Invalidate()
			return true
		case key.Matches('b', vaxis.ModCtrl):
			line := lb.selected[:lb.horizPos]
			fds := strings.Fields(line)
			if len(fds) > 1 {
				lb.moveHorizontal(
					strings.LastIndex(line,
						fds[len(fds)-1]) - lb.horizPos - 1)
			} else {
				lb.horizPos = 0
			}
			lb.Invalidate()
			return true
		case key.Matches('w', vaxis.ModCtrl):
			line := lb.selected[lb.horizPos+1:]
			fds := strings.Fields(line)
			if len(fds) > 1 {
				lb.moveHorizontal(strings.Index(line, fds[1]))
			}
			lb.Invalidate()
			return true
		case key.Matches('a', vaxis.ModCtrl), key.Matches(vaxis.KeyHome):
			if showFilter {
				break
			}
			lb.horizPos = 0
			lb.Invalidate()
			return true
		case key.Matches('e', vaxis.ModCtrl), key.Matches(vaxis.KeyEnd):
			if showFilter {
				break
			}
			lb.horizPos = len(lb.selected)
			lb.Invalidate()
			return true
		case key.Matches('p', vaxis.ModCtrl), key.Matches(vaxis.KeyUp):
			lb.moveCursor(-1)
			lb.Invalidate()
			return true
		case key.Matches('n', vaxis.ModCtrl), key.Matches(vaxis.KeyDown):
			lb.moveCursor(+1)
			lb.Invalidate()
			return true
		case key.Matches(vaxis.KeyPgUp):
			if lb.jump >= 0 {
				lb.moveCursor(-lb.jump)
				lb.Invalidate()
			}
			return true
		case key.Matches(vaxis.KeyPgDown):
			if lb.jump >= 0 {
				lb.moveCursor(+lb.jump)
				lb.Invalidate()
			}
			return true
		case key.Matches(vaxis.KeyEnter):
			return lb.quit(lb.selected)
		case key.Matches(vaxis.KeyEsc):
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
