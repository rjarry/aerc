package widgets

import (
	"time"

	tb "github.com/nsf/termbox-go"

	"git.sr.ht/~sircmpwn/aerc2/lib/ui"
)

type StatusLine struct {
	stack    []*StatusMessage
	fallback StatusMessage

	onInvalidate func(d ui.Drawable)
}

type StatusMessage struct {
	bg      tb.Attribute
	fg      tb.Attribute
	message string
}

func NewStatusLine() *StatusLine {
	return &StatusLine{
		fallback: StatusMessage{
			bg:      tb.ColorWhite,
			fg:      tb.ColorBlack,
			message: "Idle",
		},
	}
}

func (status *StatusLine) OnInvalidate(onInvalidate func (d ui.Drawable)) {
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
	cell := tb.Cell{
		Fg: line.fg,
		Bg: line.bg,
		Ch: ' ',
	}
	ctx.Fill(0, 0, ctx.Width(), ctx.Height(), cell)
	ctx.Printf(0, 0, cell, "%s", line.message)
}

func (status *StatusLine) Set(text string) *StatusMessage {
	status.fallback = StatusMessage{
		bg:      tb.ColorWhite,
		fg:      tb.ColorBlack,
		message: text,
	}
	status.Invalidate()
	return &status.fallback
}

func (status *StatusLine) Push(text string, expiry time.Duration) *StatusMessage {
	msg := &StatusMessage{
		bg:      tb.ColorWhite,
		fg:      tb.ColorBlack,
		message: text,
	}
	status.stack = append(status.stack, msg)
	go (func () {
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

func (msg *StatusMessage) Color(bg tb.Attribute, fg tb.Attribute) {
	msg.bg = bg
	msg.fg = fg
}
