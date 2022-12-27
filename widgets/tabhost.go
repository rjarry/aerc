package widgets

import (
	"time"
)

type TabHost interface {
	BeginExCommand(cmd string)
	UpdateStatus()
	SetError(err string)
	PushStatus(text string, expiry time.Duration) *StatusMessage
	PushError(text string) *StatusMessage
	PushSuccess(text string) *StatusMessage
	Beep()
}
