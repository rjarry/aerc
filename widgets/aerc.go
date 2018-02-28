package widgets

import (
	"fmt"
	"log"
	"time"

	tb "github.com/nsf/termbox-go"

	libui "git.sr.ht/~sircmpwn/aerc2/lib/ui"
)

type Aerc struct {
	grid        *libui.Grid
	tabs        *libui.Tabs
	statusbar   *libui.Stack
	statusline  *StatusLine
	interactive libui.Interactive
}

func NewAerc(logger *log.Logger) *Aerc {
	tabs := libui.NewTabs()
	tabs.Add(libui.NewFill('★'), "白い星")
	tabs.Add(libui.NewFill('☆'), "empty stars")

	grid := libui.NewGrid().Rows([]libui.GridSpec{
		libui.GridSpec{libui.SIZE_EXACT, 1},
		libui.GridSpec{libui.SIZE_WEIGHT, 1},
		libui.GridSpec{libui.SIZE_EXACT, 1},
	}).Columns([]libui.GridSpec{
		libui.GridSpec{libui.SIZE_EXACT, 20},
		libui.GridSpec{libui.SIZE_WEIGHT, 1},
	})

	// TODO: move sidebar into tab content, probably
	grid.AddChild(libui.NewText("aerc").
		Strategy(libui.TEXT_CENTER).
		Color(tb.ColorBlack, tb.ColorWhite))
	// sidebar placeholder:
	grid.AddChild(libui.NewBordered(
		libui.NewFill('.'), libui.BORDER_RIGHT)).At(1, 0).Span(2, 1)
	grid.AddChild(tabs.TabStrip).At(0, 1)
	grid.AddChild(tabs.TabContent).At(1, 1)

	statusbar := libui.NewStack()
	grid.AddChild(statusbar).At(2, 1)

	statusline := NewStatusLine()
	statusbar.Push(statusline)

	exline := NewExLine(func(command string) {
		statusline.Push(fmt.Sprintf("TODO: execute %s", command),
			3 * time.Second)
		statusbar.Pop()
	}, func() {
		statusbar.Pop()
	})
	statusbar.Push(exline)

	go (func() {
		for {
			time.Sleep(1 * time.Second)
			tabs.Select((tabs.Selected + 1) % 2)
		}
	})()

	return &Aerc{
		grid:        grid,
		interactive: exline,
		statusbar:   statusbar,
		statusline:  statusline,
		tabs:        tabs,
	}
}

func (aerc *Aerc) OnInvalidate(onInvalidate func(d libui.Drawable)) {
	aerc.grid.OnInvalidate(onInvalidate)
}

func (aerc *Aerc) Invalidate() {
	aerc.grid.Invalidate()
}

func (aerc *Aerc) Draw(ctx *libui.Context) {
	aerc.grid.Draw(ctx)
}

func (aerc *Aerc) Event(event tb.Event) bool {
	return aerc.interactive.Event(event)
}
