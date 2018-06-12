package widgets

import (
	"fmt"
	"log"
	"time"

	"github.com/gdamore/tcell"

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

	mainGrid := libui.NewGrid().Rows([]libui.GridSpec{
		{libui.SIZE_EXACT, 1},
		{libui.SIZE_WEIGHT, 1},
	}).Columns([]libui.GridSpec{
		{libui.SIZE_EXACT, 20},
		{libui.SIZE_WEIGHT, 1},
	})

	statusbar := libui.NewStack()
	statusline := NewStatusLine()
	statusbar.Push(statusline)

	// TODO: Grab sidebar size from config and via :set command
	mainGrid.AddChild(libui.NewText("aerc").
		Strategy(libui.TEXT_CENTER).
		Color(tcell.ColorBlack, tcell.ColorWhite))
	mainGrid.AddChild(tabs.TabStrip).At(0, 1)
	mainGrid.AddChild(tabs.TabContent).At(1, 0).Span(1, 2)

	acctPlaceholder := func(sidebar, body rune, name string) {
		accountGrid := libui.NewGrid().Rows([]libui.GridSpec{
			{libui.SIZE_WEIGHT, 1},
			{libui.SIZE_EXACT, 1},
		}).Columns([]libui.GridSpec{
			{libui.SIZE_EXACT, 20},
			{libui.SIZE_WEIGHT, 1},
		})
		// Sidebar placeholder
		accountGrid.AddChild(libui.NewBordered(
			libui.NewFill(sidebar), libui.BORDER_RIGHT)).Span(2, 1)
		// Message list placeholder
		accountGrid.AddChild(libui.NewFill(body)).At(0, 1)
		// Statusbar
		accountGrid.AddChild(statusbar).At(1, 1)
		tabs.Add(accountGrid, name)
	}

	acctPlaceholder('.', '★', "白い星")
	acctPlaceholder(',', '☆', "empty stars")

	go (func() {
		for {
			time.Sleep(1 * time.Second)
			tabs.Select((tabs.Selected + 1) % 2)
		}
	})()

	return &Aerc{
		grid:       mainGrid,
		statusbar:  statusbar,
		statusline: statusline,
		tabs:       tabs,
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

func (aerc *Aerc) Event(event tcell.Event) bool {
	switch event := event.(type) {
	case *tcell.EventKey:
		if event.Rune() == ':' {
			exline := NewExLine(func(command string) {
				aerc.statusline.Push(fmt.Sprintf("TODO: execute %s", command),
					3*time.Second)
				aerc.statusbar.Pop()
				aerc.interactive = nil
			}, func() {
				aerc.statusbar.Pop()
				aerc.interactive = nil
			})
			aerc.interactive = exline
			aerc.statusbar.Push(exline)
			return true
		}
	}
	if aerc.interactive != nil {
		return aerc.interactive.Event(event)
	} else {
		return false
	}
}
