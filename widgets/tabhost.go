package widgets

type TabHost interface {
	BeginExCommand(cmd string)
	SetStatus(status string) *StatusMessage
	SetError(err string) *StatusMessage
	PushStatus(text string) *StatusMessage
	PushError(text string) *StatusMessage
	PushSuccess(text string) *StatusMessage
	Beep()
}
