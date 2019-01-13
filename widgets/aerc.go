package widgets

import (
	"fmt"
	"log"
	"time"

	"github.com/gdamore/tcell"

	"git.sr.ht/~sircmpwn/aerc2/config"
	libui "git.sr.ht/~sircmpwn/aerc2/lib/ui"
)

type Aerc struct {
	grid        *libui.Grid
	tabs        *libui.Tabs
	statusbar   *libui.Stack
	statusline  *StatusLine
	interactive libui.Interactive
}

func NewAerc(conf *config.AercConfig, logger *log.Logger) *Aerc {
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

	for _, acct := range conf.Accounts {
		view, err := NewAccountView(&acct, logger, statusbar)
		if err != nil {
			// TODO: something useful (update statusline?)
			panic(err)
		}
		tabs.Add(view, acct.Name)
	}

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
