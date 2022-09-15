package widgets

import (
	"os/exec"
	"syscall"

	"git.sr.ht/~rjarry/aerc/lib/ui"
	"git.sr.ht/~rjarry/aerc/logging"
	tcellterm "git.sr.ht/~rockorager/tcell-term"

	"github.com/gdamore/tcell/v2"
	"github.com/gdamore/tcell/v2/views"
)

type Terminal struct {
	ui.Invalidatable
	closed      bool
	cmd         *exec.Cmd
	ctx         *ui.Context
	cursorShown bool
	destroyed   bool
	focus       bool
	vterm       *tcellterm.Terminal
	running     bool

	OnClose func(err error)
	OnEvent func(event tcell.Event) bool
	OnStart func()
	OnTitle func(title string)
}

func NewTerminal(cmd *exec.Cmd) (*Terminal, error) {
	term := &Terminal{
		cursorShown: true,
	}
	term.cmd = cmd
	term.vterm = tcellterm.New()
	return term, nil
}

func (term *Terminal) Close(err error) {
	if term.closed {
		return
	}
	// Stop receiving events
	term.vterm.Unwatch(term)
	if term.cmd != nil && term.cmd.Process != nil {
		err := term.cmd.Process.Kill()
		if err != nil {
			logging.Warnf("failed to kill process: %v", err)
		}
		// Race condition here, check if cmd exists. If process exits
		// fast, this could by nil and panic
		if term.cmd != nil {
			err = term.cmd.Wait()
		}
		if err != nil {
			logging.Warnf("failed for wait for process to terminate: %v", err)
		}
		term.cmd = nil
	}
	if term.vterm != nil {
		term.vterm.Close()
	}
	if !term.closed && term.OnClose != nil {
		term.OnClose(err)
	}
	if term.ctx != nil {
		term.ctx.HideCursor()
	}
	term.closed = true
}

func (term *Terminal) Destroy() {
	if term.destroyed {
		return
	}
	if term.ctx != nil {
		term.ctx.HideCursor()
	}
	// If we destroy, we don't want to call the OnClose callback
	term.OnClose = nil
	term.Close(nil)
	term.vterm = nil
	term.destroyed = true
}

func (term *Terminal) Invalidate() {
	term.invalidate()
}

func (term *Terminal) invalidate() {
	term.DoInvalidate(term)
}

func (term *Terminal) Draw(ctx *ui.Context) {
	if term.destroyed {
		return
	}
	term.ctx = ctx // gross
	term.vterm.SetView(ctx.View())
	if !term.running && !term.closed && term.cmd != nil {
		go func() {
			defer logging.PanicHandler()
			term.vterm.Watch(term)
			attr := &syscall.SysProcAttr{Setsid: true, Setctty: true, Ctty: 1}
			if err := term.vterm.RunWithAttrs(term.cmd, attr); err != nil {
				logging.Errorf("error running terminal: %w", err)
				term.Close(err)
				term.running = false
				return
			}
			term.running = false
			term.Close(nil)
		}()
		for {
			if term.cmd.Process != nil {
				term.running = true
				break
			}
		}
		if term.OnStart != nil {
			term.OnStart()
		}
	}
	term.draw()
}

func (term *Terminal) draw() {
	term.vterm.Draw()
	if term.focus && !term.closed && term.ctx != nil {
		if !term.cursorShown {
			term.ctx.HideCursor()
		} else {
			_, x, y, style := term.vterm.GetCursor()
			term.ctx.SetCursor(x, y)
			term.ctx.SetCursorStyle(style)
		}
	}
}

func (term *Terminal) MouseEvent(localX int, localY int, event tcell.Event) {
	if event, ok := event.(*tcell.EventMouse); ok {
		if term.OnEvent != nil {
			if term.OnEvent(event) {
				return
			}
		}
		if term.closed {
			return
		}
	}
}

func (term *Terminal) Focus(focus bool) {
	if term.closed {
		return
	}
	term.focus = focus
	if term.ctx != nil {
		if !term.focus {
			term.ctx.HideCursor()
		} else {
			_, x, y, style := term.vterm.GetCursor()
			term.ctx.SetCursor(x, y)
			term.ctx.SetCursorStyle(style)
			term.invalidate()
		}
	}
}

// HandleEvent is used to watch the underlying terminal events
func (term *Terminal) HandleEvent(ev tcell.Event) bool {
	if term.closed || term.destroyed {
		return false
	}
	switch ev := ev.(type) {
	case *views.EventWidgetContent:
		// Draw here for performance improvement. We call draw again in
		// the main Draw, but tcell-term only draws dirty cells, so it
		// won't be too much extra CPU there. Drawing there is needed
		// for certain msgviews, particularly if the pager command
		// exits.
		term.draw()
		// Perform a tcell screen.Show() to show our updates
		// immediately
		if term.ctx != nil {
			term.ctx.Show()
		}
		term.invalidate()
		return true
	case *tcellterm.EventTitle:
		if term.OnTitle != nil {
			term.OnTitle(ev.Title())
		}
	}
	return false
}

func (term *Terminal) Event(event tcell.Event) bool {
	if term.OnEvent != nil {
		if term.OnEvent(event) {
			return true
		}
	}
	if term.closed {
		return false
	}
	return term.vterm.HandleEvent(event)
}
