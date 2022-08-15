package widgets

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/emersion/go-message/mail"
	"github.com/gdamore/tcell/v2"
	"github.com/google/shlex"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/lib/crypto"
	"git.sr.ht/~rjarry/aerc/lib/ui"
	"git.sr.ht/~rjarry/aerc/logging"
	"git.sr.ht/~rjarry/aerc/models"
)

type Aerc struct {
	accounts    map[string]*AccountView
	cmd         func(cmd []string) error
	cmdHistory  lib.History
	complete    func(cmd string) []string
	conf        *config.AercConfig
	focused     ui.Interactive
	grid        *ui.Grid
	simulating  int
	statusbar   *ui.Stack
	statusline  *StatusLine
	pasting     bool
	pendingKeys []config.KeyStroke
	prompts     *ui.Stack
	tabs        *ui.Tabs
	ui          *ui.UI
	beep        func() error
	dialog      ui.DrawableInteractive

	Crypto crypto.Provider
}

type Choice struct {
	Key     string
	Text    string
	Command []string
}

func NewAerc(conf *config.AercConfig,
	crypto crypto.Provider, cmd func(cmd []string) error,
	complete func(cmd string) []string, cmdHistory lib.History,
	deferLoop chan struct{},
) *Aerc {
	tabs := ui.NewTabs(&conf.Ui)

	statusbar := ui.NewStack(conf.Ui)
	statusline := NewStatusLine(conf.Ui)
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

	aerc := &Aerc{
		accounts:   make(map[string]*AccountView),
		conf:       conf,
		cmd:        cmd,
		cmdHistory: cmdHistory,
		complete:   complete,
		grid:       grid,
		statusbar:  statusbar,
		statusline: statusline,
		prompts:    ui.NewStack(conf.Ui),
		tabs:       tabs,
		Crypto:     crypto,
	}

	statusline.SetAerc(aerc)
	conf.Triggers.ExecuteCommand = cmd

	for i, acct := range conf.Accounts {
		view, err := NewAccountView(aerc, conf, &conf.Accounts[i], aerc, deferLoop)
		if err != nil {
			tabs.Add(errorScreen(err.Error(), conf.Ui), acct.Name, nil)
		} else {
			aerc.accounts[acct.Name] = view
			conf := view.UiConfig()
			tabs.Add(view, acct.Name, conf)
		}
	}

	if len(conf.Accounts) == 0 {
		wizard := NewAccountWizard(aerc.Config(), aerc)
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
		case *Composer:
			aerc.RemoveTab(content)
			content.Close()
		case *Terminal:
			content.Close(nil)
		case *MessageViewer:
			aerc.RemoveTab(content)
		}
	}

	return aerc
}

func (aerc *Aerc) OnBeep(f func() error) {
	aerc.beep = f
}

func (aerc *Aerc) Beep() {
	if aerc.beep == nil {
		logging.Warnf("should beep, but no beeper")
		return
	}
	if err := aerc.beep(); err != nil {
		logging.Errorf("tried to beep, but could not: %v", err)
	}
}

func (aerc *Aerc) Tick() bool {
	more := false
	for _, acct := range aerc.accounts {
		more = acct.Tick() || more
	}

	if len(aerc.prompts.Children()) > 0 {
		more = true
		previous := aerc.focused
		prompt := aerc.prompts.Pop().(*ExLine)
		prompt.finish = func() {
			aerc.statusbar.Pop()
			aerc.focus(previous)
		}

		aerc.statusbar.Push(prompt)
		aerc.focus(prompt)
	}

	return more
}

func (aerc *Aerc) Children() []ui.Drawable {
	return aerc.grid.Children()
}

func (aerc *Aerc) OnInvalidate(onInvalidate func(d ui.Drawable)) {
	aerc.grid.OnInvalidate(func(_ ui.Drawable) {
		onInvalidate(aerc)
	})
}

func (aerc *Aerc) Invalidate() {
	aerc.grid.Invalidate()
}

func (aerc *Aerc) Focus(focus bool) {
	// who cares
}

func (aerc *Aerc) Draw(ctx *ui.Context) {
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
		s = strings.ReplaceAll(s, "<space>", " ")
		return strings.ReplaceAll(s, "%", "%%")
	}
	fmtStr := "%10s %s"
	for _, bind := range binds.Bindings {
		result = append(result, fmt.Sprintf(fmtStr,
			format(config.FormatKeyStrokes(bind.Input)),
			format(config.FormatKeyStrokes(bind.Output)),
		))
	}
	if binds.Globals && aerc.conf.Bindings.Global != nil {
		for _, bind := range aerc.conf.Bindings.Global.Bindings {
			result = append(result, fmt.Sprintf(fmtStr+" (Globals)",
				format(config.FormatKeyStrokes(bind.Input)),
				format(config.FormatKeyStrokes(bind.Output)),
			))
		}
	}
	result = append(result, fmt.Sprintf(fmtStr,
		"$ex",
		fmt.Sprintf("'%c'", binds.ExKey.Rune),
	))
	result = append(result, fmt.Sprintf(fmtStr,
		"Globals",
		fmt.Sprintf("%v", binds.Globals),
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
		binds := aerc.conf.MergeContextualBinds(aerc.conf.Bindings.MessageList, config.BIND_CONTEXT_ACCOUNT, selectedAccountName, "messages")
		return aerc.conf.MergeContextualBinds(binds, config.BIND_CONTEXT_FOLDER, view.SelectedDirectory(), "messages")
	case *AccountWizard:
		return aerc.conf.Bindings.AccountWizard
	case *Composer:
		switch view.Bindings() {
		case "compose::editor":
			return aerc.conf.MergeContextualBinds(aerc.conf.Bindings.ComposeEditor, config.BIND_CONTEXT_ACCOUNT, selectedAccountName, "compose::editor")
		case "compose::review":
			return aerc.conf.MergeContextualBinds(aerc.conf.Bindings.ComposeReview, config.BIND_CONTEXT_ACCOUNT, selectedAccountName, "compose::review")
		default:
			return aerc.conf.MergeContextualBinds(aerc.conf.Bindings.Compose, config.BIND_CONTEXT_ACCOUNT, selectedAccountName, "compose")
		}
	case *MessageViewer:
		switch view.Bindings() {
		case "view::passthrough":
			return aerc.conf.MergeContextualBinds(aerc.conf.Bindings.MessageViewPassthrough, config.BIND_CONTEXT_ACCOUNT, selectedAccountName, "view::passthrough")
		default:
			return aerc.conf.MergeContextualBinds(aerc.conf.Bindings.MessageView, config.BIND_CONTEXT_ACCOUNT, selectedAccountName, "view")
		}
	case *Terminal:
		return aerc.conf.Bindings.Terminal
	default:
		return aerc.conf.Bindings.Global
	}
}

func (aerc *Aerc) simulate(strokes []config.KeyStroke) {
	aerc.pendingKeys = []config.KeyStroke{}
	aerc.simulating += 1
	for _, stroke := range strokes {
		simulated := tcell.NewEventKey(
			stroke.Key, stroke.Rune, tcell.ModNone)
		aerc.Event(simulated)
	}
	aerc.simulating -= 1
}

func (aerc *Aerc) Event(event tcell.Event) bool {
	if aerc.dialog != nil {
		return aerc.dialog.Event(event)
	}

	if aerc.focused != nil {
		return aerc.focused.Event(event)
	}

	switch event := event.(type) {
	case *tcell.EventKey:
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
		aerc.pendingKeys = append(aerc.pendingKeys, config.KeyStroke{
			Modifiers: event.Modifiers(),
			Key:       event.Key(),
			Rune:      event.Rune(),
		})
		aerc.statusline.Invalidate()
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
			result, strokes = aerc.conf.Bindings.Global.
				GetBinding(aerc.pendingKeys)
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
				exKey = aerc.conf.Bindings.Global.ExKey
			}
			if event.Key() == exKey.Key && event.Rune() == exKey.Rune {
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

func (aerc *Aerc) Config() *config.AercConfig {
	return aerc.conf
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
		return &aerc.conf.Ui
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
	var uiConf *config.UIConfig = nil
	if acct := aerc.account(clickable); acct != nil {
		conf := acct.UiConfig()
		uiConf = conf
	}
	tab := aerc.tabs.Add(clickable, name, uiConf)
	aerc.UpdateStatus()
	return tab
}

func (aerc *Aerc) RemoveTab(tab ui.Drawable) {
	aerc.tabs.Remove(tab)
	aerc.UpdateStatus()
}

func (aerc *Aerc) ReplaceTab(tabSrc ui.Drawable, tabTarget ui.Drawable, name string) {
	aerc.tabs.Replace(tabSrc, tabTarget, name)
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

func (aerc *Aerc) TabNames() []string {
	return aerc.tabs.Names()
}

func (aerc *Aerc) SelectPreviousTab() bool {
	return aerc.tabs.SelectPrevious()
}

func (aerc *Aerc) SetStatus(status string) *StatusMessage {
	return aerc.statusline.Set(status)
}

func (aerc *Aerc) UpdateStatus() {
	if acct := aerc.SelectedAccount(); acct != nil {
		acct.UpdateStatus()
	} else {
		aerc.ClearStatus()
	}
}

func (aerc *Aerc) ClearStatus() {
	aerc.statusline.Set("")
}

func (aerc *Aerc) SetError(status string) *StatusMessage {
	return aerc.statusline.SetError(status)
}

func (aerc *Aerc) PushStatus(text string, expiry time.Duration) *StatusMessage {
	return aerc.statusline.Push(text, expiry)
}

func (aerc *Aerc) PushError(text string) *StatusMessage {
	return aerc.statusline.PushError(text)
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
	exline := NewExLine(aerc.conf, cmd, func(cmd string) {
		parts, err := shlex.Split(cmd)
		if err != nil {
			aerc.PushError(err.Error())
		}
		err = aerc.cmd(parts)
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
	}, func(cmd string) ([]string, string) {
		return aerc.complete(cmd), ""
	}, aerc.cmdHistory)
	aerc.statusbar.Push(exline)
	aerc.focus(exline)
}

func (aerc *Aerc) RegisterPrompt(prompt string, cmd []string) {
	p := NewPrompt(aerc.conf, prompt, func(text string) {
		if text != "" {
			cmd = append(cmd, text)
		}
		err := aerc.cmd(cmd)
		if err != nil {
			aerc.PushError(err.Error())
		}
	}, func(cmd string) ([]string, string) {
		return nil, "" // TODO: completions
	})
	aerc.prompts.Push(p)
}

func (aerc *Aerc) RegisterChoices(choices []Choice) {
	cmds := make(map[string][]string)
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
	p := NewPrompt(aerc.conf, prompt, func(text string) {
		cmd, ok := cmds[text]
		if !ok {
			return
		}
		err := aerc.cmd(cmd)
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
	h := &mail.Header{}
	to, err := mail.ParseAddressList(addr.Opaque)
	if err != nil && addr.Opaque != "" {
		return fmt.Errorf("Could not parse to: %w", err)
	}
	h.SetAddressList("to", to)
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

	composer, err := NewComposer(aerc, acct, aerc.Config(),
		acct.AccountConfig(), acct.Worker(), "", h, models.OriginalMail{})
	if err != nil {
		return nil
	}
	composer.SetContents(strings.NewReader(body))
	composer.FocusEditor("subject")
	title := "New email"
	if subject != "" {
		title = subject
		composer.FocusTerminal()
	}
	if to == nil {
		composer.FocusEditor("to")
	}
	tab := aerc.NewTab(composer, title)
	composer.OnHeaderChange("Subject", func(subject string) {
		if subject == "" {
			tab.Name = "New email"
		} else {
			tab.Name = subject
		}
		tab.Content.Invalidate()
	})
	return nil
}

func (aerc *Aerc) Mbox(source string) error {
	acctConf := config.AccountConfig{}
	if selectedAcct := aerc.SelectedAccount(); selectedAcct != nil {
		acctConf = *selectedAcct.acct
		info := fmt.Sprintf("Loading outgoing mbox mail settings from account [%s]", selectedAcct.Name())
		aerc.PushStatus(info, 10*time.Second)
		logging.Infof(info)
	} else {
		acctConf.From = "<user@localhost>"
	}
	acctConf.Name = "mbox"
	acctConf.Source = source
	acctConf.Default = "INBOX"
	acctConf.Archive = "Archive"
	acctConf.Postpone = "Drafts"
	acctConf.CopyTo = "Sent"

	mboxView, err := NewAccountView(aerc, aerc.conf, &acctConf, aerc, nil)
	if err != nil {
		aerc.NewTab(errorScreen(err.Error(), aerc.conf.Ui), acctConf.Name)
	} else {
		aerc.accounts[acctConf.Name] = mboxView
		aerc.NewTab(mboxView, acctConf.Name)
	}
	return nil
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
			logging.Errorf("Closing backend failed for %s: %v", acct.Name(), err)
		}
	}
	return returnErr
}

func (aerc *Aerc) AddDialog(d ui.DrawableInteractive) {
	aerc.dialog = d
	aerc.dialog.OnInvalidate(func(_ ui.Drawable) {
		aerc.Invalidate()
	})
	aerc.Invalidate()
}

func (aerc *Aerc) CloseDialog() {
	aerc.dialog = nil
	aerc.Invalidate()
}

func (aerc *Aerc) GetPassword(title string, prompt string) (chText chan string, chErr chan error) {
	chText = make(chan string, 1)
	chErr = make(chan error, 1)
	getPasswd := NewGetPasswd(title, prompt, aerc.conf, func(pw string, err error) {
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

func (aerc *Aerc) Initialize(ui *ui.UI) {
	aerc.ui = ui
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
func errorScreen(s string, conf config.UIConfig) ui.Drawable {
	errstyle := conf.GetStyle(config.STYLE_ERROR)
	text := ui.NewText(s, errstyle).Strategy(ui.TEXT_CENTER)
	grid := ui.NewGrid().Rows([]ui.GridSpec{
		{Strategy: ui.SIZE_WEIGHT, Size: ui.Const(1)},
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(1)},
		{Strategy: ui.SIZE_WEIGHT, Size: ui.Const(1)},
	}).Columns([]ui.GridSpec{
		{Strategy: ui.SIZE_WEIGHT, Size: ui.Const(1)},
	})
	grid.AddChild(ui.NewFill(' ', tcell.StyleDefault)).At(0, 0)
	grid.AddChild(text).At(1, 0)
	grid.AddChild(ui.NewFill(' ', tcell.StyleDefault)).At(2, 0)
	return grid
}
