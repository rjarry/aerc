package widgets

import (
	"time"

	"github.com/gdamore/tcell"

	"git.sr.ht/~sircmpwn/aerc2/lib/ui"
)

type StatusLine struct {
	stack    []*StatusMessage
	fallback StatusMessage

	onInvalidate func(d ui.Drawable)
}

type StatusMessage struct {
	bg      tcell.Color
	fg      tcell.Color
	message string
}

func NewStatusLine() *StatusLine {
	return &StatusLine{
		fallback: StatusMessage{
			bg:      tcell.ColorDefault,
			fg:      tcell.ColorDefault,
			message: "Idle",
		},
	}
}

func (status *StatusLine) OnInvalidate(onInvalidate func(d ui.Drawable)) {
	status.onInvalidate = onInvalidate
}

func (status *StatusLine) Invalidate() {
	if status.onInvalidate != nil {
		status.onInvalidate(status)
	}
}

func (status *StatusLine) Draw(ctx *ui.Context) {
	line := &status.fallback
	if len(status.stack) != 0 {
		line = status.stack[len(status.stack)-1]
	}
	style := tcell.StyleDefault.
		Background(line.bg).Foreground(line.fg).Reverse(true)
	ctx.Fill(0, 0, ctx.Width(), ctx.Height(), ' ', style)
	ctx.Printf(0, 0, style, "%s", line.message)
}

func (status *StatusLine) Set(text string) *StatusMessage {
	status.fallback = StatusMessage{
		bg:      tcell.ColorDefault,
		fg:      tcell.ColorDefault,
		message: text,
	}
	status.Invalidate()
	return &status.fallback
}

func (status *StatusLine) Push(text string, expiry time.Duration) *StatusMessage {
	msg := &StatusMessage{
		bg:      tcell.ColorDefault,
		fg:      tcell.ColorDefault,
		message: text,
	}
	status.stack = append(status.stack, msg)
	go (func() {
		time.Sleep(expiry)
		for i, m := range status.stack {
			if m == msg {
				status.stack = append(status.stack[:i], status.stack[i+1:]...)
				break
			}
		}
		status.Invalidate()
	})()
	return msg
}

func (status *StatusLine) Expire() {
	status.stack = nil
}

func (msg *StatusMessage) Color(bg tcell.Color, fg tcell.Color) {
	msg.bg = bg
	msg.fg = fg
}
