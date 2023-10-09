package app

import (
	"bytes"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib/state"
	"git.sr.ht/~rjarry/aerc/lib/templates"
	"git.sr.ht/~rjarry/aerc/lib/ui"
	"git.sr.ht/~rjarry/aerc/log"
)

type StatusLine struct {
	sync.Mutex
	stack []*StatusMessage
	aerc  *Aerc
	acct  *AccountView
	err   string
}

type StatusMessage struct {
	style   tcell.Style
	message string
}

func (status *StatusLine) Invalidate() {
	ui.Invalidate()
}

func (status *StatusLine) Draw(ctx *ui.Context) {
	status.Lock()
	defer status.Unlock()
	style := status.uiConfig().GetStyle(config.STYLE_STATUSLINE_DEFAULT)
	ctx.Fill(0, 0, ctx.Width(), ctx.Height(), ' ', style)
	switch {
	case len(status.stack) != 0:
		line := status.stack[len(status.stack)-1]
		msg := runewidth.Truncate(line.message, ctx.Width(), "")
		msg = runewidth.FillRight(msg, ctx.Width())
		ctx.Printf(0, 0, line.style, "%s", msg)
	case status.err != "":
		msg := runewidth.Truncate(status.err, ctx.Width(), "")
		msg = runewidth.FillRight(msg, ctx.Width())
		style := status.uiConfig().GetStyle(config.STYLE_STATUSLINE_ERROR)
		ctx.Printf(0, 0, style, "%s", msg)
	case status.aerc != nil && status.acct != nil:
		data := state.NewDataSetter()
		data.SetPendingKeys(status.aerc.pendingKeys)
		data.SetState(&status.acct.state)
		data.SetAccount(status.acct.acct)
		data.SetFolder(status.acct.Directories().SelectedDirectory())
		msg, _ := status.acct.SelectedMessage()
		data.SetInfo(msg, 0, false)
		table := ui.NewTable(
			ctx.Height(),
			config.Statusline.StatusColumns,
			config.Statusline.ColumnSeparator,
			nil,
			func(*ui.Table, int) tcell.Style { return style },
		)
		var buf bytes.Buffer
		cells := make([]string, len(table.Columns))
		for c, col := range table.Columns {
			err := templates.Render(col.Def.Template, &buf,
				data.Data())
			if err != nil {
				log.Errorf("%s", err)
				cells[c] = err.Error()
			} else {
				cells[c] = buf.String()
			}
			buf.Reset()
		}
		table.AddRow(cells, nil)
		table.Draw(ctx)
	}
}

func (status *StatusLine) Update(acct *AccountView) {
	status.acct = acct
	status.Invalidate()
}

func (status *StatusLine) SetError(err string) {
	prev := status.err
	status.err = err
	if prev != status.err {
		status.Invalidate()
	}
}

func (status *StatusLine) Clear() {
	status.SetError("")
	status.acct = nil
}

func (status *StatusLine) Push(text string, expiry time.Duration) *StatusMessage {
	status.Lock()
	defer status.Unlock()
	log.Debugf(text)
	msg := &StatusMessage{
		style:   status.uiConfig().GetStyle(config.STYLE_STATUSLINE_DEFAULT),
		message: text,
	}
	status.stack = append(status.stack, msg)
	go (func() {
		defer log.PanicHandler()

		time.Sleep(expiry)
		status.Lock()
		defer status.Unlock()
		for i, m := range status.stack {
			if m == msg {
				status.stack = append(status.stack[:i], status.stack[i+1:]...)
				break
			}
		}
		status.Invalidate()
	})()
	status.Invalidate()
	return msg
}

func (status *StatusLine) PushError(text string) *StatusMessage {
	log.Errorf(text)
	msg := status.Push(text, 10*time.Second)
	msg.Color(status.uiConfig().GetStyle(config.STYLE_STATUSLINE_ERROR))
	return msg
}

func (status *StatusLine) PushWarning(text string) *StatusMessage {
	log.Warnf(text)
	msg := status.Push(text, 10*time.Second)
	msg.Color(status.uiConfig().GetStyle(config.STYLE_STATUSLINE_WARNING))
	return msg
}

func (status *StatusLine) PushSuccess(text string) *StatusMessage {
	log.Tracef(text)
	msg := status.Push(text, 10*time.Second)
	msg.Color(status.uiConfig().GetStyle(config.STYLE_STATUSLINE_SUCCESS))
	return msg
}

func (status *StatusLine) Expire() {
	status.Lock()
	defer status.Unlock()
	status.stack = nil
}

func (status *StatusLine) uiConfig() *config.UIConfig {
	return status.aerc.SelectedAccountUiConfig()
}

func (status *StatusLine) SetAerc(aerc *Aerc) {
	status.aerc = aerc
}

func (msg *StatusMessage) Color(style tcell.Style) {
	msg.style = style
}
