package widgets

import (
	"time"

	"github.com/gdamore/tcell"
	"github.com/mattn/go-runewidth"

	"git.sr.ht/~sircmpwn/aerc/config"
	"git.sr.ht/~sircmpwn/aerc/lib/ui"
)

type StatusLine struct {
	ui.Invalidatable
	stack    []*StatusMessage
	fallback StatusMessage
	aerc     *Aerc
	uiConfig config.UIConfig
}

type StatusMessage struct {
	style   tcell.Style
	message string
}

func NewStatusLine(uiConfig config.UIConfig) *StatusLine {
	return &StatusLine{
		fallback: StatusMessage{
			style:   uiConfig.GetStyle(config.STYLE_STATUSLINE_DEFAULT),
			message: "Idle",
		},
		uiConfig: uiConfig,
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
	ctx.Fill(0, 0, ctx.Width(), ctx.Height(), ' ', line.style)
	pendingKeys := ""
	if status.aerc != nil {
		for _, pendingKey := range status.aerc.pendingKeys {
			pendingKeys += string(pendingKey.Rune)
		}
	}
	message := runewidth.FillRight(line.message, ctx.Width()-len(pendingKeys)-5)
	ctx.Printf(0, 0, line.style, "%s%s", message, pendingKeys)
}

func (status *StatusLine) Set(text string) *StatusMessage {
	status.fallback = StatusMessage{
		style:   status.uiConfig.GetStyle(config.STYLE_STATUSLINE_DEFAULT),
		message: text,
	}
	status.Invalidate()
	return &status.fallback
}

func (status *StatusLine) SetError(text string) *StatusMessage {
	status.fallback = StatusMessage{
		style:   status.uiConfig.GetStyle(config.STYLE_STATUSLINE_ERROR),
		message: text,
	}
	status.Invalidate()
	return &status.fallback
}

func (status *StatusLine) Push(text string, expiry time.Duration) *StatusMessage {
	msg := &StatusMessage{
		style:   status.uiConfig.GetStyle(config.STYLE_STATUSLINE_DEFAULT),
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

func (status *StatusLine) PushError(text string, expiry time.Duration) *StatusMessage {
	msg := status.Push(text, expiry)
	msg.Color(status.uiConfig.GetStyle(config.STYLE_STATUSLINE_ERROR))
	return msg
}

func (status *StatusLine) PushSuccess(text string, expiry time.Duration) *StatusMessage {
	msg := status.Push(text, expiry)
	msg.Color(status.uiConfig.GetStyle(config.STYLE_STATUSLINE_SUCCESS))
	return msg
}

func (status *StatusLine) Expire() {
	status.stack = nil
}

func (status *StatusLine) SetAerc(aerc *Aerc) {
	status.aerc = aerc
}

func (msg *StatusMessage) Color(style tcell.Style) {
	msg.style = style
}
