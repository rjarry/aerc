package widgets

import (
	"time"

	"github.com/gdamore/tcell"

	"git.sr.ht/~sircmpwn/aerc2/lib/ui"
)

var (
	frames = []string{
		"[..]    ",
		" [..]   ",
		"  [..]  ",
		"   [..] ",
		"    [..]",
		"   [..] ",
		"  [..]  ",
		" [..]   ",
	}
)

type Spinner struct {
	frame        int
	onInvalidate func(d ui.Drawable)
	stop         chan interface{}
}

func NewSpinner() *Spinner {
	spinner := Spinner{
		stop:  make(chan interface{}),
		frame: -1,
	}
	return &spinner
}

func (s *Spinner) Start() {
	if s.IsRunning() {
		return
	}

	s.frame = 0
	go func() {
		for {
			select {
			case <-s.stop:
				return
			case <-time.After(200 * time.Millisecond):
				s.frame++
				if s.frame >= len(frames) {
					s.frame = 0
				}
				s.Invalidate()
			}
		}
	}()
}

func (s *Spinner) Stop() {
	if !s.IsRunning() {
		return
	}

	s.stop <- nil
	s.frame = -1
	s.Invalidate()
}

func (s *Spinner) IsRunning() bool {
	return s.frame != -1
}

func (s *Spinner) Draw(ctx *ui.Context) {
	if !s.IsRunning() {
		return
	}

	ctx.Fill(0, 0, ctx.Width(), ctx.Height(), ' ', tcell.StyleDefault)
	col := ctx.Width()/2 - len(frames[0])/2 + 1
	ctx.Printf(col, 0, tcell.StyleDefault, "%s", frames[s.frame])
}

func (s *Spinner) OnInvalidate(onInvalidate func(d ui.Drawable)) {
	s.onInvalidate = onInvalidate
}

func (s *Spinner) Invalidate() {
	if s.onInvalidate != nil {
		s.onInvalidate(s)
	}
}
