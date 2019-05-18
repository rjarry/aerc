package widgets

import (
	"sync/atomic"
	"time"

	"github.com/gdamore/tcell"

	"git.sr.ht/~sircmpwn/aerc/lib/ui"
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
	ui.Invalidatable
	frame int64 // access via atomic
	stop  chan struct{}
}

func NewSpinner() *Spinner {
	spinner := Spinner{
		stop:  make(chan struct{}),
		frame: -1,
	}
	return &spinner
}

func (s *Spinner) Start() {
	if s.IsRunning() {
		return
	}

	atomic.StoreInt64(&s.frame, 0)

	go func() {
		for {
			select {
			case <-s.stop:
				atomic.StoreInt64(&s.frame, -1)
				s.stop <- struct{}{}
				return
			case <-time.After(200 * time.Millisecond):
				atomic.AddInt64(&s.frame, 1)
				s.Invalidate()
			}
		}
	}()
}

func (s *Spinner) Stop() {
	if !s.IsRunning() {
		return
	}

	s.stop <- struct{}{}
	<-s.stop
	s.Invalidate()
}

func (s *Spinner) IsRunning() bool {
	return atomic.LoadInt64(&s.frame) != -1
}

func (s *Spinner) Draw(ctx *ui.Context) {
	if !s.IsRunning() {
		s.Start()
	}

	cur := int(atomic.LoadInt64(&s.frame) % int64(len(frames)))

	ctx.Fill(0, 0, ctx.Width(), ctx.Height(), ' ', tcell.StyleDefault)
	col := ctx.Width()/2 - len(frames[0])/2 + 1
	ctx.Printf(col, 0, tcell.StyleDefault, "%s", frames[cur])
}

func (s *Spinner) Invalidate() {
	s.DoInvalidate(s)
}
