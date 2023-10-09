package app

import (
	"os/exec"
	"sync/atomic"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib/ui"
	"git.sr.ht/~rjarry/aerc/log"
	tcellterm "git.sr.ht/~rockorager/tcell-term"

	"github.com/gdamore/tcell/v2"
)

type Terminal struct {
	closed  int32
	cmd     *exec.Cmd
	ctx     *ui.Context
	focus   bool
	visible bool
	vterm   *tcellterm.VT
	running bool

	OnClose func(err error)
	OnEvent func(event tcell.Event) bool
	OnStart func()
	OnTitle func(title string)
}

func NewTerminal(cmd *exec.Cmd) (*Terminal, error) {
	term := &Terminal{
		cmd:     cmd,
		vterm:   tcellterm.New(),
		visible: true,
	}
	term.vterm.OSC8 = config.General.EnableOSC8
	term.vterm.TERM = config.General.Term
	return term, nil
}

func (term *Terminal) Close() {
	term.closeErr(nil)
}

// TODO: replace with atomic.Bool when min go version will have it (1.19+)
const closed int32 = 1

func (term *Terminal) isClosed() bool {
	return atomic.LoadInt32(&term.closed) == closed
}

func (term *Terminal) closeErr(err error) {
	if atomic.SwapInt32(&term.closed, closed) == closed {
		return
	}
	if term.vterm != nil {
		// Stop receiving events
		term.vterm.Detach()
		term.vterm.Close()
	}
	if term.OnClose != nil {
		term.OnClose(err)
	}
	ui.Invalidate()
}

func (term *Terminal) Destroy() {
	// If we destroy, we don't want to call the OnClose callback
	term.OnClose = nil
	term.closeErr(nil)
}

func (term *Terminal) Invalidate() {
	ui.Invalidate()
}

func (term *Terminal) Draw(ctx *ui.Context) {
	term.vterm.SetSurface(ctx.View())

	w, h := ctx.View().Size()
	if !term.isClosed() && term.ctx != nil {
		ow, oh := term.ctx.View().Size()
		if w != ow || h != oh {
			term.vterm.Resize(w, h)
		}
	}
	term.ctx = ctx
	if !term.running && term.cmd != nil {
		term.vterm.Attach(term.HandleEvent)
		if err := term.vterm.Start(term.cmd); err != nil {
			log.Errorf("error running terminal: %v", err)
			term.closeErr(err)
			return
		}
		term.running = true
		if term.OnStart != nil {
			term.OnStart()
		}
	}
	term.vterm.Draw()
	if term.focus {
		y, x, style, vis := term.vterm.Cursor()
		if vis && !term.isClosed() {
			ctx.SetCursor(x, y)
			ctx.SetCursorStyle(style)
		} else {
			ctx.HideCursor()
		}
	}
}

func (term *Terminal) Show(visible bool) {
	term.visible = visible
}

func (term *Terminal) MouseEvent(localX int, localY int, event tcell.Event) {
	ev, ok := event.(*tcell.EventMouse)
	if !ok {
		return
	}
	if term.OnEvent != nil {
		term.OnEvent(ev)
	}
	if term.isClosed() {
		return
	}
	e := tcell.NewEventMouse(localX, localY, ev.Buttons(), ev.Modifiers())
	term.vterm.HandleEvent(e)
}

func (term *Terminal) Focus(focus bool) {
	if term.isClosed() {
		return
	}
	term.focus = focus
	if term.ctx != nil {
		if !term.focus {
			term.ctx.HideCursor()
		} else {
			y, x, style, _ := term.vterm.Cursor()
			term.ctx.SetCursor(x, y)
			term.ctx.SetCursorStyle(style)
			term.Invalidate()
		}
	}
}

// HandleEvent is used to watch the underlying terminal events
func (term *Terminal) HandleEvent(ev tcell.Event) {
	if term.isClosed() {
		return
	}
	switch ev := ev.(type) {
	case *tcellterm.EventRedraw:
		if term.visible {
			ui.Invalidate()
		}
	case *tcellterm.EventTitle:
		if term.OnTitle != nil {
			term.OnTitle(ev.Title())
		}
	case *tcellterm.EventClosed:
		term.Close()
		ui.Invalidate()
	}
}

func (term *Terminal) Event(event tcell.Event) bool {
	if term.OnEvent != nil {
		if term.OnEvent(event) {
			return true
		}
	}
	if term.isClosed() {
		return false
	}
	return term.vterm.HandleEvent(event)
}
