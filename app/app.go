package app

import (
	"time"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/lib/crypto"
	"git.sr.ht/~rjarry/aerc/lib/ipc"
	"git.sr.ht/~rjarry/aerc/lib/ui"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
	"github.com/ProtonMail/go-crypto/openpgp"
)

var aerc *Aerc

func Init(
	crypto crypto.Provider,
	cmd func([]string, *config.AccountConfig, *models.MessageInfo) error,
	complete func(cmd string) ([]string, string), history lib.History,
	deferLoop chan struct{},
) {
	aerc = NewAerc(crypto, cmd, complete, history, deferLoop)
}

func Drawable() ui.DrawableInteractive      { return aerc }
func IPCHandler() ipc.Handler               { return aerc }
func HandleMessage(msg types.WorkerMessage) { aerc.HandleMessage(msg) }

func CloseBackends() error { return aerc.CloseBackends() }

func AddDialog(d ui.DrawableInteractive) { aerc.AddDialog(d) }
func CloseDialog()                       { aerc.CloseDialog() }

func HumanReadableBindings() []string {
	return aerc.HumanReadableBindings()
}

func Account(name string) (*AccountView, error) { return aerc.Account(name) }
func AccountNames() []string                    { return aerc.AccountNames() }
func NextAccount() (*AccountView, error)        { return aerc.NextAccount() }
func PrevAccount() (*AccountView, error)        { return aerc.PrevAccount() }
func SelectedAccount() *AccountView             { return aerc.SelectedAccount() }
func SelectedAccountUiConfig() *config.UIConfig { return aerc.SelectedAccountUiConfig() }

func NextTab()                                          { aerc.NextTab() }
func PrevTab()                                          { aerc.PrevTab() }
func PinTab()                                           { aerc.PinTab() }
func UnpinTab()                                         { aerc.UnpinTab() }
func MoveTab(i int, relative bool)                      { aerc.MoveTab(i, relative) }
func TabNames() []string                                { return aerc.TabNames() }
func SelectTab(name string) bool                        { return aerc.SelectTab(name) }
func SelectPreviousTab() bool                           { return aerc.SelectPreviousTab() }
func SelectedTab() *ui.Tab                              { return aerc.SelectedTab() }
func SelectedTabContent() ui.Drawable                   { return aerc.SelectedTabContent() }
func SelectTabIndex(index int) bool                     { return aerc.SelectTabIndex(index) }
func RemoveTab(tab ui.Drawable, closeContent bool)      { aerc.RemoveTab(tab, closeContent) }
func NewTab(clickable ui.Drawable, name string) *ui.Tab { return aerc.NewTab(clickable, name) }
func ReplaceTab(tabSrc ui.Drawable, tabTarget ui.Drawable, name string, closeSrc bool) {
	aerc.ReplaceTab(tabSrc, tabTarget, name, closeSrc)
}

func UpdateStatus()                          { aerc.UpdateStatus() }
func PushPrompt(prompt *ExLine)              { aerc.PushPrompt(prompt) }
func SetError(text string)                   { aerc.SetError(text) }
func PushError(text string) *StatusMessage   { return aerc.PushError(text) }
func PushWarning(text string) *StatusMessage { return aerc.PushWarning(text) }
func PushSuccess(text string) *StatusMessage { return aerc.PushSuccess(text) }
func PushStatus(text string, expiry time.Duration) *StatusMessage {
	return aerc.PushStatus(text, expiry)
}

func RegisterChoices(choices []Choice)           { aerc.RegisterChoices(choices) }
func RegisterPrompt(prompt string, cmd []string) { aerc.RegisterPrompt(prompt, cmd) }

func CryptoProvider() crypto.Provider { return aerc.Crypto }
func DecryptKeys(keys []openpgp.Key, symmetric bool) (b []byte, err error) {
	return aerc.DecryptKeys(keys, symmetric)
}
