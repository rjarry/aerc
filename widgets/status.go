package widgets

import (
	"time"

	"github.com/gdamore/tcell"
	"github.com/mattn/go-runewidth"

	"git.sr.ht/~sircmpwn/aerc/lib/ui"
)

type StatusLine struct {
	ui.Invalidatable
	stack    []*StatusMessage
	fallback StatusMessage
	aerc     *Aerc
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

func (status *StatusLine) Invalidate() {
	status.DoInvalidate(status)
}

func (status *StatusLine) Draw(ctx *ui.Context) {
	line := &status.fallback
	if len(status.stack) != 0 {
		line = status.stack[len(status.stack)-1]
	}
	style := tcell.StyleDefault.
		Background(line.bg).Foreground(line.fg).Reverse(true)
	ctx.Fill(0, 0, ctx.Width(), ctx.Height(), ' ', style)
	pendingKeys := ""
	if status.aerc != nil {
		for _, pendingKey := range status.aerc.pendingKeys {
			pendingKeys += string(pendingKey.Rune)
		}
	}
	message := runewidth.FillRight(line.message, ctx.Width()-len(pendingKeys)-5)
	ctx.Printf(0, 0, style, "%s%s", message, pendingKeys)
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

func (status *StatusLine) SetAerc(aerc *Aerc) {
	status.aerc = aerc
}

func (msg *StatusMessage) Color(bg tcell.Color, fg tcell.Color) {
	msg.bg = bg
	msg.fg = fg
}
