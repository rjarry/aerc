package app

import (
	"bytes"
	"os/exec"
	"sync/atomic"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/lib/state"
	"git.sr.ht/~rjarry/aerc/lib/ui"
	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/widgets/term"
)

type HasTerminal interface {
	Terminal() *Terminal
}

type Terminal struct {
	// Bell was rung in the terminal since it last had focus.
	bell    bool
	closed  int32
	visible int32 // visible if >0
	cmd     *exec.Cmd
	ctx     *ui.Context
	focus   bool
	vterm   *term.Model
	running bool
	tab     *ui.Tab
	// The window title set by the application running in the terminal.
	title string

	OnClose func(err error)
	OnEvent func(event vaxis.Event) bool
	OnStart func()
	OnTitle func(title string)
}

func NewTerminal(cmd *exec.Cmd) (*Terminal, error) {
	term := &Terminal{
		cmd:     cmd,
		vterm:   term.New(),
		visible: 1,
	}
	term.vterm.OSC8 = config.General().EnableOSC8
	term.vterm.TERM = config.General().Term
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
		if term.ctx != nil {
			term.ctx.HideCursor()
		}
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
	if ctx.Width() == 0 || ctx.Height() == 0 {
		return
	}
	term.ctx = ctx
	if !term.running && term.cmd != nil {
		term.vterm.Attach(term.HandleEvent)
		w, h := ctx.Window().Size()
		if err := term.vterm.StartWithSize(term.cmd, w, h); err != nil {
			log.Errorf("error running terminal: %v", err)
			term.closeErr(err)
			return
		}
		term.running = true
		if term.OnStart != nil {
			term.OnStart()
		}
	}
	term.vterm.Draw(ctx.Window())
}

func (term *Terminal) Show(visible bool) {
	if visible {
		atomic.StoreInt32(&term.visible, 1)
	} else {
		atomic.StoreInt32(&term.visible, 0)
	}
}

func (term *Terminal) Terminal() *Terminal {
	return term
}

func (term *Terminal) MouseEvent(localX int, localY int, event vaxis.Event) {
	ev, ok := event.(vaxis.Mouse)
	if !ok {
		return
	}
	if term.OnEvent != nil {
		term.OnEvent(ev)
	}
	if term.isClosed() {
		return
	}
	ev.Row = localY
	ev.Col = localX
	term.vterm.Update(ev)
}

func (term *Terminal) Focus(focus bool) {
	if term.isClosed() {
		return
	}
	term.focus = focus
	term.bell = false
	if term.focus {
		term.vterm.Focus()
	} else {
		term.vterm.Blur()
	}
	term.setTitle()
	ui.Invalidate()
}

// HandleEvent is used to watch the underlying terminal events
func (t *Terminal) HandleEvent(ev vaxis.Event) {
	if t.isClosed() {
		return
	}
	switch ev := ev.(type) {
	case vaxis.Redraw:
		if atomic.LoadInt32(&t.visible) > 0 {
			ui.Invalidate()
		}
	case term.EventTitle:
		t.title = string(ev)
		t.setTitle()
		ui.Invalidate()
		if t.OnTitle != nil {
			t.OnTitle(string(ev))
		}
	case term.EventClosed:
		t.Close()
		ui.Invalidate()
	case term.EventBell:
		if !t.focus {
			t.bell = true
			t.setTitle()
			ui.Invalidate()
		}
		aerc.Beep()
	}
}

func (term *Terminal) Event(event vaxis.Event) bool {
	if term.OnEvent != nil {
		if term.OnEvent(event) {
			return true
		}
	}
	if term.isClosed() {
		return false
	}
	term.vterm.Update(event)
	return true
}

// Renders the terminal tab title based on the template, falls back to the
// unformatted title in case of errors.
func (t *Terminal) RenderTitle() string {
	title := t.title
	if title == "" {
		title = t.cmd.Path
	}

	data := state.NewDataSetter()
	data.SetTitle(title)
	data.SetBell(t.bell)
	var buf bytes.Buffer
	if err := config.Ui().TabTitleTerminal.Execute(&buf, data); err != nil {
		PushError("failed to render terminal tab title: " + err.Error())
		return title
	}
	return buf.String()
}

func (t *Terminal) SetTab(tab *ui.Tab) {
	t.tab = tab
}

func (t *Terminal) setTitle() {
	if t.tab != nil {
		t.tab.SetTitle(t.RenderTitle())
	}
}
