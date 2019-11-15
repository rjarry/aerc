package widgets

import (
	"time"
)

type TabHost interface {
	BeginExCommand(cmd string)
	SetStatus(status string) *StatusMessage
	PushStatus(text string, expiry time.Duration) *StatusMessage
	Beep()
}
