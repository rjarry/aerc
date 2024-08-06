package app

import (
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/lib/ui"
	"git.sr.ht/~rockorager/vaxis"
	"github.com/riywo/loginshell"
)

var qt quakeTerminal

type quakeTerminal struct {
	mu      sync.Mutex
	rolling int32
	visible bool
	term    *Terminal
}

func ToggleQuake() {
	handleErr := func(err error) {
		log.Errorf("quake-terminal: %v", err)
	}
	if !qt.HasTerm() {
		shell, err := loginshell.Shell()
		if err != nil {
			handleErr(err)
			return
		}
		args := []string{shell}
		cmd := exec.Command(args[0], args[1:]...)
		term, err := NewTerminal(cmd)
		if err != nil {
			handleErr(err)
			return
		}
		term.OnClose = func(err error) {
			if err != nil {
				aerc.PushError(err.Error())
			}
			qt.Hide()
			qt.SetTerm(nil)
		}
		qt.SetTerm(term)
	}

	if qt.Rolling() {
		return
	}

	if qt.Visible() {
		qt.Hide()
	} else {
		qt.Show()
	}
}

func (q *quakeTerminal) Rolling() bool {
	return atomic.LoadInt32(&q.rolling) > 0
}

func (q *quakeTerminal) SetTerm(t *Terminal) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.term = t
}

func (q *quakeTerminal) HasTerm() bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	return q.term != nil
}

func (q *quakeTerminal) Visible() bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	return q.visible
}

// inputReturn is helper function to create dialog boxes.
func inputReturn() func(int) int {
	return func(x int) int { return x }
}

// fixReturn is helper function to create dialog boxes.
func fixReturn(x int) func(int) int {
	return func(_ int) int { return x }
}

func (q *quakeTerminal) Show() {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.term == nil {
		return
	}

	uiConfig := SelectedAccountUiConfig()
	h := uiConfig.QuakeHeight

	termBox := NewDialog(
		ui.NewBox(q.term, "", "", uiConfig),
		fixReturn(0),
		fixReturn(0),
		inputReturn(),
		fixReturn(h),
	)

	f := Roller{
		span: 100 * time.Millisecond,
		done: func() {
			log.Tracef("restore after show")
			atomic.StoreInt32(&q.rolling, 0)
			ui.QueueFunc(func() {
				CloseDialog()
				AddDialog(termBox)
			})
		},
	}

	atomic.StoreInt32(&q.rolling, 1)
	emptyBox := NewDialog(
		ui.NewBox(&EmptyInteractive{}, "", "", uiConfig),
		fixReturn(0),
		fixReturn(0),
		inputReturn(),
		f.Roll(1, h),
	)

	q.visible = true
	if q.term != nil {
		q.term.Show(q.visible)
		q.term.Focus(q.visible)
	}

	CloseDialog()
	AddDialog(emptyBox)
}

func (q *quakeTerminal) Hide() {
	uiConfig := SelectedAccountUiConfig()
	f := Roller{
		span: 100 * time.Millisecond,
		done: func() {
			atomic.StoreInt32(&q.rolling, 0)
			ui.QueueFunc(CloseDialog)
			log.Tracef("restore after hide")
		},
	}

	atomic.StoreInt32(&q.rolling, 1)
	emptyBox := NewDialog(
		ui.NewBox(&EmptyInteractive{}, "", "", uiConfig),
		fixReturn(0),
		fixReturn(0),
		inputReturn(),
		f.Roll(uiConfig.QuakeHeight, 2),
	)

	q.mu.Lock()
	q.visible = false
	if q.term != nil {
		q.term.Focus(q.visible)
		q.term.Show(q.visible)
	}
	q.mu.Unlock()

	ui.QueueFunc(func() {
		CloseDialog()
		AddDialog(emptyBox)
	})
}

type EmptyInteractive struct{}

func (e *EmptyInteractive) Draw(ctx *ui.Context) {
	w := ctx.Width()
	h := ctx.Height()
	if w == 0 || h == 0 {
		return
	}
	style := SelectedAccountUiConfig().GetStyle(config.STYLE_DEFAULT)
	ctx.Fill(0, 0, ctx.Width(), ctx.Height(), ' ', style)
}

func (e *EmptyInteractive) Invalidate() {
}

func (e *EmptyInteractive) MouseEvent(_ int, _ int, _ vaxis.Event) {
}

func (e *EmptyInteractive) Event(_ vaxis.Event) bool {
	return true
}

func (e *EmptyInteractive) Focus(_ bool) {
}

type Roller struct {
	span  time.Duration
	done  func()
	value int64
}

func (f *Roller) Roll(start, end int) func(int) int {
	nsteps := end - start

	var step int64 = 1
	if end < start {
		step = -1
		nsteps = -nsteps
	}

	span := f.span.Milliseconds() / int64(nsteps)
	refresh := time.Duration(span) * time.Millisecond

	atomic.StoreInt64(&f.value, int64(start))

	go func() {
		defer log.PanicHandler()
		for i := 0; i < int(nsteps); i++ {
			aerc.Invalidate()
			time.Sleep(refresh)
			atomic.AddInt64(&f.value, step)
		}
		if f.done != nil {
			ui.QueueFunc(f.done)
		}
	}()

	return func(_ int) int {
		log.Tracef("in roller")
		return int(atomic.LoadInt64(&f.value))
	}
}
