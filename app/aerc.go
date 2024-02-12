package app

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"os/exec"
	"sort"
	"strings"
	"time"
	"unicode"

	"git.sr.ht/~rjarry/go-opt"
	"git.sr.ht/~rockorager/vaxis"
	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/emersion/go-message/mail"
	"github.com/gdamore/tcell/v2"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/lib/crypto"
	"git.sr.ht/~rjarry/aerc/lib/ui"
	"git.sr.ht/~rjarry/aerc/log"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

type Aerc struct {
	accounts    map[string]*AccountView
	cmd         func(string, *config.AccountConfig, *models.MessageInfo) error
	cmdHistory  lib.History
	complete    func(cmd string) ([]string, string)
	focused     ui.Interactive
	grid        *ui.Grid
	simulating  int
	statusbar   *ui.Stack
	statusline  *StatusLine
	pasting     bool
	pendingKeys []config.KeyStroke
	prompts     *ui.Stack
	tabs        *ui.Tabs
	beep        func() error
	dialog      ui.DrawableInteractive

	Crypto crypto.Provider
}

type Choice struct {
	Key     string
	Text    string
	Command string
}

func (aerc *Aerc) Init(
	crypto crypto.Provider,
	cmd func(string, *config.AccountConfig, *models.MessageInfo) error,
	complete func(cmd string) ([]string, string), cmdHistory lib.History,
	deferLoop chan struct{},
) {
	tabs := ui.NewTabs(config.Ui)

	statusbar := ui.NewStack(config.Ui)
	statusline := &StatusLine{}
	statusbar.Push(statusline)

	grid := ui.NewGrid().Rows([]ui.GridSpec{
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(1)},
		{Strategy: ui.SIZE_WEIGHT, Size: ui.Const(1)},
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(1)},
	}).Columns([]ui.GridSpec{
		{Strategy: ui.SIZE_WEIGHT, Size: ui.Const(1)},
	})
	grid.AddChild(tabs.TabStrip)
	grid.AddChild(tabs.TabContent).At(1, 0)
	grid.AddChild(statusbar).At(2, 0)

	aerc.accounts = make(map[string]*AccountView)
	aerc.cmd = cmd
	aerc.cmdHistory = cmdHistory
	aerc.complete = complete
	aerc.grid = grid
	aerc.statusbar = statusbar
	aerc.statusline = statusline
	aerc.prompts = ui.NewStack(config.Ui)
	aerc.tabs = tabs
	aerc.Crypto = crypto

	for _, acct := range config.Accounts {
		view, err := NewAccountView(acct, deferLoop)
		if err != nil {
			tabs.Add(errorScreen(err.Error()), acct.Name, nil)
		} else {
			aerc.accounts[acct.Name] = view
			view.tab = tabs.Add(view, acct.Name, view.UiConfig())
		}
	}

	if len(config.Accounts) == 0 {
		wizard := NewAccountWizard()
		wizard.Focus(true)
		aerc.NewTab(wizard, "New account")
	}

	tabs.Select(0)

	tabs.CloseTab = func(index int) {
		tab := aerc.tabs.Get(index)
		if tab == nil {
			return
		}
		switch content := tab.Content.(type) {
		case *AccountView:
			return
		case *AccountWizard:
			return
		default:
			aerc.RemoveTab(content, true)
		}
	}

	aerc.showConfigWarnings()
}

func (aerc *Aerc) showConfigWarnings() {
	var dialogs []ui.DrawableInteractive

	callback := func(string, error) {
		aerc.CloseDialog()
		if len(dialogs) > 0 {
			d := dialogs[0]
			dialogs = dialogs[1:]
			aerc.AddDialog(d)
		}
	}

	for _, w := range config.Warnings {
		dialogs = append(dialogs, NewSelectorDialog(
			w.Title, w.Body, []string{"OK"}, 0,
			aerc.SelectedAccountUiConfig(),
			callback,
		))
	}

	callback("", nil)
}

func (aerc *Aerc) OnBeep(f func() error) {
	aerc.beep = f
}

func (aerc *Aerc) Beep() {
	if aerc.beep == nil {
		log.Warnf("should beep, but no beeper")
		return
	}
	if err := aerc.beep(); err != nil {
		log.Errorf("tried to beep, but could not: %v", err)
	}
}

func (aerc *Aerc) HandleMessage(msg types.WorkerMessage) {
	if acct, ok := aerc.accounts[msg.Account()]; ok {
		acct.onMessage(msg)
	}
}

func (aerc *Aerc) Invalidate() {
	ui.Invalidate()
}

func (aerc *Aerc) Focus(focus bool) {
	// who cares
}

func (aerc *Aerc) Draw(ctx *ui.Context) {
	if len(aerc.prompts.Children()) > 0 {
		previous := aerc.focused
		prompt := aerc.prompts.Pop().(*ExLine)
		prompt.finish = func() {
			aerc.statusbar.Pop()
			aerc.focus(previous)
		}

		aerc.statusbar.Push(prompt)
		aerc.focus(prompt)
	}
	aerc.grid.Draw(ctx)
	if aerc.dialog != nil {
		if w, h := ctx.Width(), ctx.Height(); w > 8 && h > 4 {
			if d, ok := aerc.dialog.(Dialog); ok {
				start, height := d.ContextHeight()
				aerc.dialog.Draw(
					ctx.Subcontext(4, start(h),
						w-8, height(h)))
			} else {
				aerc.dialog.Draw(ctx.Subcontext(4, h/2-2, w-8, 4))
			}
		}
	}
}

func (aerc *Aerc) HumanReadableBindings() []string {
	var result []string
	binds := aerc.getBindings()
	format := func(s string) string {
		return strings.ReplaceAll(s, "%", "%%")
	}
	annotate := func(b *config.Binding) string {
		if b.Annotation == "" {
			return ""
		}
		return "[" + b.Annotation + "]"
	}
	fmtStr := "%10s %s %s"
	for _, bind := range binds.Bindings {
		result = append(result, fmt.Sprintf(fmtStr,
			format(config.FormatKeyStrokes(bind.Input)),
			format(config.FormatKeyStrokes(bind.Output)),
			annotate(bind),
		))
	}
	if binds.Globals && config.Binds.Global != nil {
		for _, bind := range config.Binds.Global.Bindings {
			result = append(result, fmt.Sprintf(fmtStr+" (Globals)",
				format(config.FormatKeyStrokes(bind.Input)),
				format(config.FormatKeyStrokes(bind.Output)),
				annotate(bind),
			))
		}
	}
	result = append(result, fmt.Sprintf(fmtStr,
		"$ex",
		fmt.Sprintf("'%c'", binds.ExKey.Key), "",
	))
	result = append(result, fmt.Sprintf(fmtStr,
		"Globals",
		fmt.Sprintf("%v", binds.Globals), "",
	))
	sort.Strings(result)
	return result
}

func (aerc *Aerc) getBindings() *config.KeyBindings {
	selectedAccountName := ""
	if aerc.SelectedAccount() != nil {
		selectedAccountName = aerc.SelectedAccount().acct.Name
	}
	switch view := aerc.SelectedTabContent().(type) {
	case *AccountView:
		binds := config.Binds.MessageList.ForAccount(selectedAccountName)
		return binds.ForFolder(view.SelectedDirectory())
	case *AccountWizard:
		return config.Binds.AccountWizard
	case *Composer:
		var binds *config.KeyBindings
		switch view.Bindings() {
		case "compose::editor":
			binds = config.Binds.ComposeEditor.ForAccount(
				selectedAccountName)
		case "compose::review":
			binds = config.Binds.ComposeReview.ForAccount(
				selectedAccountName)
		default:
			binds = config.Binds.Compose.ForAccount(
				selectedAccountName)
		}
		return binds.ForFolder(view.SelectedDirectory())
	case *MessageViewer:
		switch view.Bindings() {
		case "view::passthrough":
			return config.Binds.MessageViewPassthrough.ForAccount(
				selectedAccountName)
		default:
			return config.Binds.MessageView.ForAccount(
				selectedAccountName)
		}
	case *Terminal:
		return config.Binds.Terminal
	default:
		return config.Binds.Global
	}
}

func (aerc *Aerc) simulate(strokes []config.KeyStroke) {
	aerc.pendingKeys = []config.KeyStroke{}
	bindings := aerc.getBindings()
	complete := aerc.SelectedAccountUiConfig().CompletionMinChars != config.MANUAL_COMPLETE
	aerc.simulating += 1

	for _, stroke := range strokes {
		simulated := vaxis.Key{
			Keycode:   stroke.Key,
			Modifiers: stroke.Modifiers,
		}
		if unicode.IsUpper(stroke.Key) {
			simulated.Keycode = unicode.ToLower(stroke.Key)
			simulated.Modifiers |= vaxis.ModShift
		}
		// If none of these mods are present, set the text field to
		// enable matching keys like ":"
		if stroke.Modifiers&vaxis.ModCtrl == 0 &&
			stroke.Modifiers&vaxis.ModAlt == 0 &&
			stroke.Modifiers&vaxis.ModSuper == 0 &&
			stroke.Modifiers&vaxis.ModHyper == 0 {

			simulated.Text = string(stroke.Key)
		}
		aerc.Event(simulated)
		complete = stroke == bindings.CompleteKey
	}
	aerc.simulating -= 1
	if exline, ok := aerc.focused.(*ExLine); ok {
		// we are still focused on the exline, turn on tab complete
		exline.TabComplete(func(cmd string) ([]string, string) {
			return aerc.complete(cmd)
		})
		if complete {
			// force completion now
			exline.Event(vaxis.Key{Keycode: vaxis.KeyTab})
		}
	}
}

func (aerc *Aerc) Event(event vaxis.Event) bool {
	if aerc.dialog != nil {
		return aerc.dialog.Event(event)
	}

	if aerc.focused != nil {
		return aerc.focused.Event(event)
	}

	switch event := event.(type) {
	// TODO: more vaxis events handling
	case vaxis.Key:
		// If we are in a bracketed paste, don't process the keys for
		// bindings
		if aerc.pasting {
			interactive, ok := aerc.SelectedTabContent().(ui.Interactive)
			if ok {
				return interactive.Event(event)
			}
			return false
		}
		aerc.statusline.Expire()
		stroke := config.KeyStroke{
			Modifiers: event.Modifiers,
		}
		switch {
		case event.ShiftedCode != 0:
			stroke.Key = event.ShiftedCode
			stroke.Modifiers &^= vaxis.ModShift
		default:
			stroke.Key = event.Keycode
		}
		aerc.pendingKeys = append(aerc.pendingKeys, stroke)
		ui.Invalidate()
		bindings := aerc.getBindings()
		incomplete := false
		result, strokes := bindings.GetBinding(aerc.pendingKeys)
		switch result {
		case config.BINDING_FOUND:
			aerc.simulate(strokes)
			return true
		case config.BINDING_INCOMPLETE:
			incomplete = true
		case config.BINDING_NOT_FOUND:
		}
		if bindings.Globals {
			result, strokes = config.Binds.Global.GetBinding(aerc.pendingKeys)
			switch result {
			case config.BINDING_FOUND:
				aerc.simulate(strokes)
				return true
			case config.BINDING_INCOMPLETE:
				incomplete = true
			case config.BINDING_NOT_FOUND:
			}
		}
		if !incomplete {
			aerc.pendingKeys = []config.KeyStroke{}
			exKey := bindings.ExKey
			if aerc.simulating > 0 {
				// Keybindings still use : even if you change the ex key
				exKey = config.Binds.Global.ExKey
			}
			if aerc.isExKey(event, exKey) {
				aerc.BeginExCommand("")
				return true
			}
			interactive, ok := aerc.SelectedTabContent().(ui.Interactive)
			if ok {
				return interactive.Event(event)
			}
			return false
		}
	case *tcell.EventMouse:
		x, y := event.Position()
		aerc.grid.MouseEvent(x, y, event)
		return true
	case *tcell.EventPaste:
		if event.Start() {
			aerc.pasting = true
		}
		if event.End() {
			aerc.pasting = false
		}
		interactive, ok := aerc.SelectedTabContent().(ui.Interactive)
		if ok {
			return interactive.Event(event)
		}
		return false
	}
	return false
}

func (aerc *Aerc) SelectedAccount() *AccountView {
	return aerc.account(aerc.SelectedTabContent())
}

func (aerc *Aerc) Account(name string) (*AccountView, error) {
	if acct, ok := aerc.accounts[name]; ok {
		return acct, nil
	}
	return nil, fmt.Errorf("account <%s> not found", name)
}

func (aerc *Aerc) PrevAccount() (*AccountView, error) {
	cur := aerc.SelectedAccount()
	if cur == nil {
		return nil, fmt.Errorf("no account selected, cannot get prev")
	}
	for i, conf := range config.Accounts {
		if conf.Name == cur.Name() {
			i -= 1
			if i == -1 {
				i = len(config.Accounts) - 1
			}
			conf = config.Accounts[i]
			return aerc.Account(conf.Name)
		}
	}
	return nil, fmt.Errorf("no prev account")
}

func (aerc *Aerc) NextAccount() (*AccountView, error) {
	cur := aerc.SelectedAccount()
	if cur == nil {
		return nil, fmt.Errorf("no account selected, cannot get next")
	}
	for i, conf := range config.Accounts {
		if conf.Name == cur.Name() {
			i += 1
			if i == len(config.Accounts) {
				i = 0
			}
			conf = config.Accounts[i]
			return aerc.Account(conf.Name)
		}
	}
	return nil, fmt.Errorf("no next account")
}

func (aerc *Aerc) AccountNames() []string {
	results := make([]string, 0)
	for name := range aerc.accounts {
		results = append(results, name)
	}
	return results
}

func (aerc *Aerc) account(d ui.Drawable) *AccountView {
	switch tab := d.(type) {
	case *AccountView:
		return tab
	case *MessageViewer:
		return tab.SelectedAccount()
	case *Composer:
		return tab.Account()
	}
	return nil
}

func (aerc *Aerc) SelectedAccountUiConfig() *config.UIConfig {
	acct := aerc.SelectedAccount()
	if acct == nil {
		return config.Ui
	}
	return acct.UiConfig()
}

func (aerc *Aerc) SelectedTabContent() ui.Drawable {
	tab := aerc.tabs.Selected()
	if tab == nil {
		return nil
	}
	return tab.Content
}

func (aerc *Aerc) SelectedTab() *ui.Tab {
	return aerc.tabs.Selected()
}

func (aerc *Aerc) NewTab(clickable ui.Drawable, name string) *ui.Tab {
	uiConf := config.Ui
	if acct := aerc.account(clickable); acct != nil {
		uiConf = acct.UiConfig()
	}
	tab := aerc.tabs.Add(clickable, name, uiConf)
	aerc.UpdateStatus()
	return tab
}

func (aerc *Aerc) RemoveTab(tab ui.Drawable, closeContent bool) {
	aerc.tabs.Remove(tab)
	aerc.UpdateStatus()
	if content, ok := tab.(ui.Closeable); ok && closeContent {
		content.Close()
	}
}

func (aerc *Aerc) ReplaceTab(tabSrc ui.Drawable, tabTarget ui.Drawable, name string, closeSrc bool) {
	aerc.tabs.Replace(tabSrc, tabTarget, name)
	if content, ok := tabSrc.(ui.Closeable); ok && closeSrc {
		content.Close()
	}
}

func (aerc *Aerc) MoveTab(i int, relative bool) {
	aerc.tabs.MoveTab(i, relative)
}

func (aerc *Aerc) PinTab() {
	aerc.tabs.PinTab()
}

func (aerc *Aerc) UnpinTab() {
	aerc.tabs.UnpinTab()
}

func (aerc *Aerc) NextTab() {
	aerc.tabs.NextTab()
}

func (aerc *Aerc) PrevTab() {
	aerc.tabs.PrevTab()
}

func (aerc *Aerc) SelectTab(name string) bool {
	ok := aerc.tabs.SelectName(name)
	if ok {
		aerc.UpdateStatus()
	}
	return ok
}

func (aerc *Aerc) SelectTabIndex(index int) bool {
	ok := aerc.tabs.Select(index)
	if ok {
		aerc.UpdateStatus()
	}
	return ok
}

func (aerc *Aerc) SelectTabAtOffset(offset int) {
	aerc.tabs.SelectOffset(offset)
}

func (aerc *Aerc) TabNames() []string {
	return aerc.tabs.Names()
}

func (aerc *Aerc) SelectPreviousTab() bool {
	return aerc.tabs.SelectPrevious()
}

func (aerc *Aerc) UpdateStatus() {
	if acct := aerc.SelectedAccount(); acct != nil {
		aerc.statusline.Update(acct)
	} else {
		aerc.statusline.Clear()
	}
}

func (aerc *Aerc) SetError(err string) {
	aerc.statusline.SetError(err)
}

func (aerc *Aerc) PushStatus(text string, expiry time.Duration) *StatusMessage {
	return aerc.statusline.Push(text, expiry)
}

func (aerc *Aerc) PushError(text string) *StatusMessage {
	return aerc.statusline.PushError(text)
}

func (aerc *Aerc) PushWarning(text string) *StatusMessage {
	return aerc.statusline.PushWarning(text)
}

func (aerc *Aerc) PushSuccess(text string) *StatusMessage {
	return aerc.statusline.PushSuccess(text)
}

func (aerc *Aerc) focus(item ui.Interactive) {
	if aerc.focused == item {
		return
	}
	if aerc.focused != nil {
		aerc.focused.Focus(false)
	}
	aerc.focused = item
	interactive, ok := aerc.SelectedTabContent().(ui.Interactive)
	if item != nil {
		item.Focus(true)
		if ok {
			interactive.Focus(false)
		}
	} else if ok {
		interactive.Focus(true)
	}
}

func (aerc *Aerc) BeginExCommand(cmd string) {
	previous := aerc.focused
	var tabComplete func(string) ([]string, string)
	if aerc.simulating != 0 {
		// Don't try to draw completions for simulated events
		tabComplete = nil
	} else {
		tabComplete = func(cmd string) ([]string, string) {
			return aerc.complete(cmd)
		}
	}
	exline := NewExLine(cmd, func(cmd string) {
		err := aerc.cmd(cmd, nil, nil)
		if err != nil {
			aerc.PushError(err.Error())
		}
		// only add to history if this is an unsimulated command,
		// ie one not executed from a keybinding
		if aerc.simulating == 0 {
			aerc.cmdHistory.Add(cmd)
		}
	}, func() {
		aerc.statusbar.Pop()
		aerc.focus(previous)
	}, tabComplete, aerc.cmdHistory)
	aerc.statusbar.Push(exline)
	aerc.focus(exline)
}

func (aerc *Aerc) PushPrompt(prompt *ExLine) {
	aerc.prompts.Push(prompt)
}

func (aerc *Aerc) RegisterPrompt(prompt string, cmd string) {
	p := NewPrompt(prompt, func(text string) {
		if text != "" {
			cmd += " " + opt.QuoteArg(text)
		}
		err := aerc.cmd(cmd, nil, nil)
		if err != nil {
			aerc.PushError(err.Error())
		}
	}, func(cmd string) ([]string, string) {
		return nil, "" // TODO: completions
	})
	aerc.prompts.Push(p)
}

func (aerc *Aerc) RegisterChoices(choices []Choice) {
	cmds := make(map[string]string)
	texts := []string{}
	for _, c := range choices {
		text := fmt.Sprintf("[%s] %s", c.Key, c.Text)
		if strings.Contains(c.Text, c.Key) {
			text = strings.Replace(c.Text, c.Key, "["+c.Key+"]", 1)
		}
		texts = append(texts, text)
		cmds[c.Key] = c.Command
	}
	prompt := strings.Join(texts, ", ") + "? "
	p := NewPrompt(prompt, func(text string) {
		cmd, ok := cmds[text]
		if !ok {
			return
		}
		err := aerc.cmd(cmd, nil, nil)
		if err != nil {
			aerc.PushError(err.Error())
		}
	}, func(cmd string) ([]string, string) {
		return nil, "" // TODO: completions
	})
	aerc.prompts.Push(p)
}

func (aerc *Aerc) Mailto(addr *url.URL) error {
	var subject string
	var body string
	var acctName string
	var attachments []string
	h := &mail.Header{}
	to, err := mail.ParseAddressList(addr.Opaque)
	if err != nil && addr.Opaque != "" {
		return fmt.Errorf("Could not parse to: %w", err)
	}
	h.SetAddressList("to", to)
	template := config.Templates.NewMessage
	for key, vals := range addr.Query() {
		switch strings.ToLower(key) {
		case "account":
			acctName = strings.Join(vals, "")
		case "bcc":
			list, err := mail.ParseAddressList(strings.Join(vals, ","))
			if err != nil {
				break
			}
			h.SetAddressList("Bcc", list)
		case "body":
			body = strings.Join(vals, "\n")
		case "cc":
			list, err := mail.ParseAddressList(strings.Join(vals, ","))
			if err != nil {
				break
			}
			h.SetAddressList("Cc", list)
		case "in-reply-to":
			for i, msgID := range vals {
				if len(msgID) > 1 && msgID[0] == '<' &&
					msgID[len(msgID)-1] == '>' {
					vals[i] = msgID[1 : len(msgID)-1]
				}
			}
			h.SetMsgIDList("In-Reply-To", vals)
		case "subject":
			subject = strings.Join(vals, ",")
			h.SetText("Subject", subject)
		case "template":
			template = strings.Join(vals, "")
			log.Tracef("template set to %s", template)
		case "attach":
			for _, path := range vals {
				// remove a potential file:// prefix.
				attachments = append(attachments, strings.TrimPrefix(path, "file://"))
			}
		default:
			// any other header gets ignored on purpose to avoid control headers
			// being injected
		}
	}

	acct := aerc.SelectedAccount()
	if acctName != "" {
		if a, ok := aerc.accounts[acctName]; ok && a != nil {
			acct = a
		}
	}

	if acct == nil {
		return errors.New("No account selected")
	}

	defer ui.Invalidate()

	composer, err := NewComposer(acct,
		acct.AccountConfig(), acct.Worker(),
		config.Compose.EditHeaders, template, h, nil,
		strings.NewReader(body))
	if err != nil {
		return err
	}
	composer.FocusEditor("subject")
	title := "New email"
	if subject != "" {
		title = subject
		composer.FocusTerminal()
	}
	if to == nil {
		composer.FocusEditor("to")
	}
	composer.Tab = aerc.NewTab(composer, title)

	for _, file := range attachments {
		composer.AddAttachment(file)
	}
	return nil
}

func (aerc *Aerc) Mbox(source string) error {
	acctConf := config.AccountConfig{}
	if selectedAcct := aerc.SelectedAccount(); selectedAcct != nil {
		acctConf = *selectedAcct.acct
		info := fmt.Sprintf("Loading outgoing mbox mail settings from account [%s]", selectedAcct.Name())
		aerc.PushStatus(info, 10*time.Second)
		log.Debugf(info)
	} else {
		acctConf.From = &mail.Address{Address: "user@localhost"}
	}
	acctConf.Name = "mbox"
	acctConf.Source = source
	acctConf.Default = "INBOX"
	acctConf.Archive = "Archive"
	acctConf.Postpone = "Drafts"
	acctConf.CopyTo = "Sent"

	defer ui.Invalidate()

	mboxView, err := NewAccountView(&acctConf, nil)
	if err != nil {
		aerc.NewTab(errorScreen(err.Error()), acctConf.Name)
	} else {
		aerc.accounts[acctConf.Name] = mboxView
		aerc.NewTab(mboxView, acctConf.Name)
	}
	return nil
}

func (aerc *Aerc) Command(cmd string) error {
	defer ui.Invalidate()
	return aerc.cmd(cmd, nil, nil)
}

func (aerc *Aerc) CloseBackends() error {
	var returnErr error
	for _, acct := range aerc.accounts {
		var raw interface{} = acct.worker.Backend
		c, ok := raw.(io.Closer)
		if !ok {
			continue
		}
		err := c.Close()
		if err != nil {
			returnErr = err
			log.Errorf("Closing backend failed for %s: %v", acct.Name(), err)
		}
	}
	return returnErr
}

func (aerc *Aerc) AddDialog(d ui.DrawableInteractive) {
	aerc.dialog = d
	aerc.Invalidate()
}

func (aerc *Aerc) CloseDialog() {
	aerc.dialog = nil
	aerc.Invalidate()
}

func (aerc *Aerc) GetPassword(title string, prompt string) (chText chan string, chErr chan error) {
	chText = make(chan string, 1)
	chErr = make(chan error, 1)
	getPasswd := NewGetPasswd(title, prompt, func(pw string, err error) {
		defer func() {
			close(chErr)
			close(chText)
			aerc.CloseDialog()
		}()
		if err != nil {
			chErr <- err
			return
		}
		chErr <- nil
		chText <- pw
	})
	aerc.AddDialog(getPasswd)

	return
}

func (aerc *Aerc) DecryptKeys(keys []openpgp.Key, symmetric bool) (b []byte, err error) {
	for _, key := range keys {
		ident := key.Entity.PrimaryIdentity()
		chPass, chErr := aerc.GetPassword("Decrypt PGP private key",
			fmt.Sprintf("Enter password for %s (%8X)\nPress <ESC> to cancel",
				ident.Name, key.PublicKey.KeyId))

		for err := range chErr {
			if err != nil {
				return nil, err
			}
			pass := <-chPass
			err = key.PrivateKey.Decrypt([]byte(pass))
			return nil, err
		}
	}
	return nil, err
}

// errorScreen is a widget that draws an error in the middle of the context
func errorScreen(s string) ui.Drawable {
	errstyle := config.Ui.GetStyle(config.STYLE_ERROR)
	text := ui.NewText(s, errstyle).Strategy(ui.TEXT_CENTER)
	grid := ui.NewGrid().Rows([]ui.GridSpec{
		{Strategy: ui.SIZE_WEIGHT, Size: ui.Const(1)},
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(1)},
		{Strategy: ui.SIZE_WEIGHT, Size: ui.Const(1)},
	}).Columns([]ui.GridSpec{
		{Strategy: ui.SIZE_WEIGHT, Size: ui.Const(1)},
	})
	grid.AddChild(ui.NewFill(' ', vaxis.Style{})).At(0, 0)
	grid.AddChild(text).At(1, 0)
	grid.AddChild(ui.NewFill(' ', vaxis.Style{})).At(2, 0)
	return grid
}

func (aerc *Aerc) isExKey(key vaxis.Key, exKey config.KeyStroke) bool {
	return key.Matches(exKey.Key, exKey.Modifiers)
}

// CmdFallbackSearch checks cmds for the first executable availabe in PATH. An error is
// returned if none are found
func CmdFallbackSearch(cmds []string, silent bool) (string, error) {
	var tried []string
	for _, cmd := range cmds {
		if cmd == "" {
			continue
		}
		params := strings.Split(cmd, " ")
		_, err := exec.LookPath(params[0])
		if err != nil {
			tried = append(tried, cmd)
			if !silent {
				warn := fmt.Sprintf("cmd '%s' not found in PATH, using fallback", cmd)
				PushWarning(warn)
			}
			continue
		}
		return cmd, nil
	}
	return "", fmt.Errorf("no command found in PATH: %s", tried)
}
