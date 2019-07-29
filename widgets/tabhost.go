package widgets

import (
	"time"
)

type TabHost interface {
	BeginExCommand()
	SetStatus(status string) *StatusMessage
	PushStatus(text string, expiry time.Duration) *StatusMessage
	Beep()
}
