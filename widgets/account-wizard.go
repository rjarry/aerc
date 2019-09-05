package widgets

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell"
	"github.com/go-ini/ini"
	"github.com/kyoh86/xdg"

	"git.sr.ht/~sircmpwn/aerc/config"
	"git.sr.ht/~sircmpwn/aerc/lib/ui"
)

const (
	CONFIGURE_BASICS   = iota
	CONFIGURE_INCOMING = iota
	CONFIGURE_OUTGOING = iota
	CONFIGURE_COMPLETE = iota
)

const (
	IMAP_OVER_TLS = iota
	IMAP_STARTTLS = iota
	IMAP_INSECURE = iota
)

const (
	SMTP_OVER_TLS = iota
	SMTP_STARTTLS = iota
	SMTP_INSECURE = iota
)

type AccountWizard struct {
	ui.Invalidatable
	aerc      *Aerc
	conf      *config.AercConfig
	step      int
	steps     []*ui.Grid
	focus     int
	temporary bool
	testing   bool
	// CONFIGURE_BASICS
	accountName *ui.TextInput
	email       *ui.TextInput
	fullName    *ui.TextInput
	basics      []ui.Interactive
	// CONFIGURE_INCOMING
	imapUsername *ui.TextInput
	imapPassword *ui.TextInput
	imapServer   *ui.TextInput
	imapMode     int
	imapStr      *ui.Text
	imapUrl      url.URL
	incoming     []ui.Interactive
	// CONFIGURE_OUTGOING
	smtpUsername *ui.TextInput
	smtpPassword *ui.TextInput
	smtpServer   *ui.TextInput
	smtpMode     int
	smtpStr      *ui.Text
	smtpUrl      url.URL
	copySent     bool
	outgoing     []ui.Interactive
	// CONFIGURE_COMPLETE
	complete []ui.Interactive
}

func NewAccountWizard(conf *config.AercConfig, aerc *Aerc) *AccountWizard {
	wizard := &AccountWizard{
		accountName:  ui.NewTextInput("").Prompt("> "),
		aerc:         aerc,
		conf:         conf,
		temporary:    false,
		copySent:     true,
		email:        ui.NewTextInput("").Prompt("> "),
		fullName:     ui.NewTextInput("").Prompt("> "),
		imapPassword: ui.NewTextInput("").Prompt("] ").Password(true),
		imapServer:   ui.NewTextInput("").Prompt("> "),
		imapStr:      ui.NewText("imaps://"),
		imapUsername: ui.NewTextInput("").Prompt("> "),
		smtpPassword: ui.NewTextInput("").Prompt("] ").Password(true),
		smtpServer:   ui.NewTextInput("").Prompt("> "),
		smtpStr:      ui.NewText("smtps://"),
		smtpUsername: ui.NewTextInput("").Prompt("> "),
	}

	// Autofill some stuff for the user
	wizard.email.OnChange(func(_ *ui.TextInput) {
		value := wizard.email.String()
		wizard.imapUsername.Set(value)
		wizard.smtpUsername.Set(value)
		if strings.ContainsRune(value, '@') {
			server := value[strings.IndexRune(value, '@')+1:]
			wizard.imapServer.Set(server)
			wizard.smtpServer.Set(server)
		}
		wizard.imapUri()
		wizard.smtpUri()
	})
	wizard.imapServer.OnChange(func(_ *ui.TextInput) {
		imapServerURI := wizard.imapServer.String()
		smtpServerURI := imapServerURI
		if strings.HasPrefix(imapServerURI, "imap.") {
			smtpServerURI = strings.Replace(imapServerURI, "imap.", "smtp.", 1)
		}
		wizard.smtpServer.Set(smtpServerURI)
		wizard.imapUri()
		wizard.smtpUri()
	})
	wizard.imapUsername.OnChange(func(_ *ui.TextInput) {
		wizard.smtpUsername.Set(wizard.imapUsername.String())
		wizard.imapUri()
		wizard.smtpUri()
	})
	wizard.imapPassword.OnChange(func(_ *ui.TextInput) {
		wizard.smtpPassword.Set(wizard.imapPassword.String())
		wizard.imapUri()
		wizard.smtpUri()
	})
	wizard.smtpServer.OnChange(func(_ *ui.TextInput) {
		wizard.smtpUri()
	})
	wizard.smtpUsername.OnChange(func(_ *ui.TextInput) {
		wizard.smtpUri()
	})
	wizard.smtpPassword.OnChange(func(_ *ui.TextInput) {
		wizard.smtpUri()
	})

	basics := ui.NewGrid().Rows([]ui.GridSpec{
		{ui.SIZE_EXACT, 8}, // Introduction
		{ui.SIZE_EXACT, 1}, // Account name (label)
		{ui.SIZE_EXACT, 1}, // (input)
		{ui.SIZE_EXACT, 1}, // Padding
		{ui.SIZE_EXACT, 1}, // Full name (label)
		{ui.SIZE_EXACT, 1}, // (input)
		{ui.SIZE_EXACT, 1}, // Padding
		{ui.SIZE_EXACT, 1}, // Email address (label)
		{ui.SIZE_EXACT, 1}, // (input)
		{ui.SIZE_WEIGHT, 1},
	}).Columns([]ui.GridSpec{
		{ui.SIZE_WEIGHT, 1},
	})
	basics.AddChild(
		ui.NewText("\nWelcome to aerc! Let's configure your account.\n\n" +
			"This wizard supports basic IMAP & SMTP configuration.\n" +
			"For other configurations, use <Ctrl+q> to exit and read the " +
			"aerc-config(5) man page.\n" +
			"Press <Tab> and <Shift+Tab> to cycle between each field in this form, or <Ctrl+j> and <Ctrl+k>."))
	basics.AddChild(
		ui.NewText("Name for this account? (e.g. 'Personal' or 'Work')").
			Bold(true)).
		At(1, 0)
	basics.AddChild(wizard.accountName).
		At(2, 0)
	basics.AddChild(ui.NewFill(' ')).
		At(3, 0)
	basics.AddChild(
		ui.NewText("Full name for outgoing emails? (e.g. 'John Doe')").
			Bold(true)).
		At(4, 0)
	basics.AddChild(wizard.fullName).
		At(5, 0)
	basics.AddChild(ui.NewFill(' ')).
		At(6, 0)
	basics.AddChild(
		ui.NewText("Your email address? (e.g. 'john@example.org')").Bold(true)).
		At(7, 0)
	basics.AddChild(wizard.email).
		At(8, 0)
	selecter := newSelecter([]string{"Next"}, 0).
		OnChoose(func(option string) {
			email := wizard.email.String()
			if strings.ContainsRune(email, '@') {
				server := email[strings.IndexRune(email, '@')+1:]
				hostport, srv := getSRV(server, []string{"imaps", "imap"})
				if hostport != "" {
					wizard.imapServer.Set(hostport)
					if srv == "imaps" {
						wizard.imapMode = IMAP_OVER_TLS
					} else {
						wizard.imapMode = IMAP_STARTTLS
					}
					wizard.imapUri()
				}
				hostport, srv = getSRV(server, []string{"submission"})
				if hostport != "" {
					wizard.smtpServer.Set(hostport)
					wizard.smtpMode = SMTP_STARTTLS
					wizard.smtpUri()
				}
			}
			wizard.advance(option)
		})
	basics.AddChild(selecter).At(9, 0)
	wizard.basics = []ui.Interactive{
		wizard.accountName, wizard.fullName, wizard.email, selecter,
	}
	basics.OnInvalidate(func(_ ui.Drawable) {
		wizard.Invalidate()
	})

	incoming := ui.NewGrid().Rows([]ui.GridSpec{
		{ui.SIZE_EXACT, 3}, // Introduction
		{ui.SIZE_EXACT, 1}, // Username (label)
		{ui.SIZE_EXACT, 1}, // (input)
		{ui.SIZE_EXACT, 1}, // Padding
		{ui.SIZE_EXACT, 1}, // Password (label)
		{ui.SIZE_EXACT, 1}, // (input)
		{ui.SIZE_EXACT, 1}, // Padding
		{ui.SIZE_EXACT, 1}, // Server (label)
		{ui.SIZE_EXACT, 1}, // (input)
		{ui.SIZE_EXACT, 1}, // Padding
		{ui.SIZE_EXACT, 1}, // Connection mode (label)
		{ui.SIZE_EXACT, 2}, // (input)
		{ui.SIZE_EXACT, 1}, // Padding
		{ui.SIZE_EXACT, 1}, // Connection string
		{ui.SIZE_WEIGHT, 1},
	}).Columns([]ui.GridSpec{
		{ui.SIZE_WEIGHT, 1},
	})
	incoming.AddChild(ui.NewText("\nConfigure incoming mail (IMAP)"))
	incoming.AddChild(
		ui.NewText("Username").Bold(true)).
		At(1, 0)
	incoming.AddChild(wizard.imapUsername).
		At(2, 0)
	incoming.AddChild(ui.NewFill(' ')).
		At(3, 0)
	incoming.AddChild(
		ui.NewText("Password").Bold(true)).
		At(4, 0)
	incoming.AddChild(wizard.imapPassword).
		At(5, 0)
	incoming.AddChild(ui.NewFill(' ')).
		At(6, 0)
	incoming.AddChild(
		ui.NewText("Server address "+
			"(e.g. 'mail.example.org' or 'mail.example.org:1313')").Bold(true)).
		At(7, 0)
	incoming.AddChild(wizard.imapServer).
		At(8, 0)
	incoming.AddChild(ui.NewFill(' ')).
		At(9, 0)
	incoming.AddChild(
		ui.NewText("Connection mode").Bold(true)).
		At(10, 0)
	imapMode := newSelecter([]string{
		"IMAP over SSL/TLS",
		"IMAP with STARTTLS",
		"Insecure IMAP",
	}, 0).Chooser(true).OnSelect(func(option string) {
		switch option {
		case "IMAP over SSL/TLS":
			wizard.imapMode = IMAP_OVER_TLS
		case "IMAP with STARTTLS":
			wizard.imapMode = IMAP_STARTTLS
		case "Insecure IMAP":
			wizard.imapMode = IMAP_INSECURE
		}
		wizard.imapUri()
	})
	incoming.AddChild(imapMode).At(11, 0)
	selecter = newSelecter([]string{"Previous", "Next"}, 1).
		OnChoose(wizard.advance)
	incoming.AddChild(ui.NewFill(' ')).At(12, 0)
	incoming.AddChild(wizard.imapStr).At(13, 0)
	incoming.AddChild(selecter).At(14, 0)
	wizard.incoming = []ui.Interactive{
		wizard.imapUsername, wizard.imapPassword, wizard.imapServer,
		imapMode, selecter,
	}
	incoming.OnInvalidate(func(_ ui.Drawable) {
		wizard.Invalidate()
	})

	outgoing := ui.NewGrid().Rows([]ui.GridSpec{
		{ui.SIZE_EXACT, 3}, // Introduction
		{ui.SIZE_EXACT, 1}, // Username (label)
		{ui.SIZE_EXACT, 1}, // (input)
		{ui.SIZE_EXACT, 1}, // Padding
		{ui.SIZE_EXACT, 1}, // Password (label)
		{ui.SIZE_EXACT, 1}, // (input)
		{ui.SIZE_EXACT, 1}, // Padding
		{ui.SIZE_EXACT, 1}, // Server (label)
		{ui.SIZE_EXACT, 1}, // (input)
		{ui.SIZE_EXACT, 1}, // Padding
		{ui.SIZE_EXACT, 1}, // Connection mode (label)
		{ui.SIZE_EXACT, 2}, // (input)
		{ui.SIZE_EXACT, 1}, // Padding
		{ui.SIZE_EXACT, 1}, // Connection string
		{ui.SIZE_EXACT, 1}, // Padding
		{ui.SIZE_EXACT, 1}, // Copy to sent (label)
		{ui.SIZE_EXACT, 2}, // (input)
		{ui.SIZE_WEIGHT, 1},
	}).Columns([]ui.GridSpec{
		{ui.SIZE_WEIGHT, 1},
	})
	outgoing.AddChild(ui.NewText("\nConfigure outgoing mail (SMTP)"))
	outgoing.AddChild(
		ui.NewText("Username").Bold(true)).
		At(1, 0)
	outgoing.AddChild(wizard.smtpUsername).
		At(2, 0)
	outgoing.AddChild(ui.NewFill(' ')).
		At(3, 0)
	outgoing.AddChild(
		ui.NewText("Password").Bold(true)).
		At(4, 0)
	outgoing.AddChild(wizard.smtpPassword).
		At(5, 0)
	outgoing.AddChild(ui.NewFill(' ')).
		At(6, 0)
	outgoing.AddChild(
		ui.NewText("Server address "+
			"(e.g. 'mail.example.org' or 'mail.example.org:1313')").Bold(true)).
		At(7, 0)
	outgoing.AddChild(wizard.smtpServer).
		At(8, 0)
	outgoing.AddChild(ui.NewFill(' ')).
		At(9, 0)
	outgoing.AddChild(
		ui.NewText("Connection mode").Bold(true)).
		At(10, 0)
	smtpMode := newSelecter([]string{
		"SMTP over SSL/TLS",
		"SMTP with STARTTLS",
		"Insecure SMTP",
	}, 0).Chooser(true).OnSelect(func(option string) {
		switch option {
		case "SMTP over SSL/TLS":
			wizard.smtpMode = SMTP_OVER_TLS
		case "SMTP with STARTTLS":
			wizard.smtpMode = SMTP_STARTTLS
		case "Insecure SMTP":
			wizard.smtpMode = SMTP_INSECURE
		}
		wizard.smtpUri()
	})
	outgoing.AddChild(smtpMode).At(11, 0)
	selecter = newSelecter([]string{"Previous", "Next"}, 1).
		OnChoose(wizard.advance)
	outgoing.AddChild(ui.NewFill(' ')).At(12, 0)
	outgoing.AddChild(wizard.smtpStr).At(13, 0)
	outgoing.AddChild(ui.NewFill(' ')).At(14, 0)
	outgoing.AddChild(
		ui.NewText("Copy sent messages to 'Sent' folder?").Bold(true)).
		At(15, 0)
	copySent := newSelecter([]string{"Yes", "No"}, 0).
		Chooser(true).OnChoose(func(option string) {
		switch option {
		case "Yes":
			wizard.copySent = true
		case "No":
			wizard.copySent = false
		}
	})
	outgoing.AddChild(copySent).At(16, 0)
	outgoing.AddChild(selecter).At(17, 0)
	wizard.outgoing = []ui.Interactive{
		wizard.smtpUsername, wizard.smtpPassword, wizard.smtpServer,
		smtpMode, copySent, selecter,
	}
	outgoing.OnInvalidate(func(_ ui.Drawable) {
		wizard.Invalidate()
	})

	complete := ui.NewGrid().Rows([]ui.GridSpec{
		{ui.SIZE_EXACT, 7},  // Introduction
		{ui.SIZE_WEIGHT, 1}, // Previous / Finish / Finish & open tutorial
	}).Columns([]ui.GridSpec{
		{ui.SIZE_WEIGHT, 1},
	})
	complete.AddChild(ui.NewText(
		"\nConfiguration complete!\n\n" +
			"You can go back and double check your settings, or choose 'Finish' to\n" +
			"save your settings to accounts.conf.\n\n" +
			"To add another account in the future, run ':new-account'."))
	selecter = newSelecter([]string{
		"Previous",
		"Finish & open tutorial",
		"Finish",
	}, 1).OnChoose(func(option string) {
		switch option {
		case "Previous":
			wizard.advance("Previous")
		case "Finish & open tutorial":
			wizard.finish(true)
		case "Finish":
			wizard.finish(false)
		}
	})
	complete.AddChild(selecter).At(1, 0)
	wizard.complete = []ui.Interactive{selecter}
	complete.OnInvalidate(func(_ ui.Drawable) {
		wizard.Invalidate()
	})

	wizard.steps = []*ui.Grid{basics, incoming, outgoing, complete}
	return wizard
}

func (wizard *AccountWizard) ConfigureTemporaryAccount(temporary bool) {
	wizard.temporary = temporary
}

func (wizard *AccountWizard) errorFor(d ui.Interactive, err error) {
	if d == nil {
		wizard.aerc.PushStatus(" "+err.Error(), 10*time.Second).
			Color(tcell.ColorDefault, tcell.ColorRed)
		wizard.Invalidate()
		return
	}
	for step, interactives := range [][]ui.Interactive{
		wizard.basics,
		wizard.incoming,
		wizard.outgoing,
	} {
		for focus, item := range interactives {
			if item == d {
				wizard.Focus(false)
				wizard.step = step
				wizard.focus = focus
				wizard.Focus(true)
				wizard.aerc.PushStatus(" "+err.Error(), 10*time.Second).
					Color(tcell.ColorDefault, tcell.ColorRed)
				wizard.Invalidate()
				return
			}
		}
	}
}

func (wizard *AccountWizard) finish(tutorial bool) {
	accountsConf := path.Join(xdg.ConfigHome(), "aerc", "accounts.conf")

	// Validation
	if wizard.accountName.String() == "" {
		wizard.errorFor(wizard.accountName,
			errors.New("Account name is required"))
		return
	}
	if wizard.email.String() == "" {
		wizard.errorFor(wizard.email,
			errors.New("Email address is required"))
		return
	}
	if wizard.fullName.String() == "" {
		wizard.errorFor(wizard.fullName,
			errors.New("Full name is required"))
		return
	}
	if wizard.imapServer.String() == "" {
		wizard.errorFor(wizard.imapServer,
			errors.New("IMAP server is required"))
		return
	}
	if wizard.imapServer.String() == "" {
		wizard.errorFor(wizard.smtpServer,
			errors.New("SMTP server is required"))
		return
	}

	file, err := ini.Load(accountsConf)
	if err != nil {
		file = ini.Empty()
	}

	var sec *ini.Section
	if sec, _ = file.GetSection(wizard.accountName.String()); sec != nil {
		wizard.errorFor(wizard.accountName,
			errors.New("An account by this name already exists"))
		return
	}
	sec, _ = file.NewSection(wizard.accountName.String())
	sec.NewKey("source", wizard.imapUrl.String())
	sec.NewKey("outgoing", wizard.smtpUrl.String())
	sec.NewKey("default", "INBOX")
	if wizard.smtpMode == SMTP_STARTTLS {
		sec.NewKey("smtp-starttls", "yes")
	}
	sec.NewKey("from", fmt.Sprintf("%s <%s>",
		wizard.fullName.String(), wizard.email.String()))
	if wizard.copySent {
		sec.NewKey("copy-to", "Sent")
	}

	if !wizard.temporary {
		f, err := os.OpenFile(accountsConf, os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			wizard.errorFor(nil, err)
			return
		}
		if _, err = file.WriteTo(f); err != nil {
			wizard.errorFor(nil, err)
			return
		}
	}

	account := config.AccountConfig{
		Name:     sec.Name(),
		Default:  "INBOX",
		From:     sec.Key("from").String(),
		Source:   sec.Key("source").String(),
		Outgoing: sec.Key("outgoing").String(),
	}
	if wizard.smtpMode == SMTP_STARTTLS {
		account.Params = map[string]string{
			"smtp-starttls": "yes",
		}
	}
	if wizard.copySent {
		account.CopyTo = "Sent"
	}
	wizard.conf.Accounts = append(wizard.conf.Accounts, account)

	view := NewAccountView(wizard.aerc, wizard.conf, &account,
		wizard.aerc.logger, wizard.aerc)
	wizard.aerc.accounts[account.Name] = view
	wizard.aerc.NewTab(view, account.Name)

	if tutorial {
		name := "aerc-tutorial"
		if _, err := os.Stat("./aerc-tutorial.7"); !os.IsNotExist(err) {
			// For development
			name = "./aerc-tutorial.7"
		}
		term, err := NewTerminal(exec.Command("man", name))
		if err != nil {
			wizard.errorFor(nil, err)
			return
		}
		wizard.aerc.NewTab(term, "Tutorial")
		term.OnClose = func(err error) {
			wizard.aerc.RemoveTab(term)
			if err != nil {
				wizard.aerc.PushStatus(" "+err.Error(), 10*time.Second).
					Color(tcell.ColorDefault, tcell.ColorRed)
			}
		}
	}

	wizard.aerc.RemoveTab(wizard)
}

func (wizard *AccountWizard) imapUri() url.URL {
	host := wizard.imapServer.String()
	user := wizard.imapUsername.String()
	pass := wizard.imapPassword.String()
	var scheme string
	switch wizard.imapMode {
	case IMAP_OVER_TLS:
		scheme = "imaps"
	case IMAP_STARTTLS:
		scheme = "imap"
	case IMAP_INSECURE:
		scheme = "imap+insecure"
	}
	var (
		userpass   *url.Userinfo
		userwopass *url.Userinfo
	)
	if pass == "" {
		userpass = url.User(user)
		userwopass = userpass
	} else {
		userpass = url.UserPassword(user, pass)
		userwopass = url.UserPassword(user, strings.Repeat("*", len(pass)))
	}
	uri := url.URL{
		Scheme: scheme,
		Host:   host,
		User:   userpass,
	}
	clean := url.URL{
		Scheme: scheme,
		Host:   host,
		User:   userwopass,
	}
	wizard.imapStr.Text("Connection URL: " +
		strings.ReplaceAll(clean.String(), "%2A", "*"))
	wizard.imapUrl = uri
	return uri
}

func (wizard *AccountWizard) smtpUri() url.URL {
	host := wizard.smtpServer.String()
	user := wizard.smtpUsername.String()
	pass := wizard.smtpPassword.String()
	var scheme string
	switch wizard.smtpMode {
	case SMTP_OVER_TLS:
		scheme = "smtps+plain"
	case SMTP_STARTTLS:
		scheme = "smtp+plain"
	case SMTP_INSECURE:
		scheme = "smtp+plain"
	}
	var (
		userpass   *url.Userinfo
		userwopass *url.Userinfo
	)
	if pass == "" {
		userpass = url.User(user)
		userwopass = userpass
	} else {
		userpass = url.UserPassword(user, pass)
		userwopass = url.UserPassword(user, strings.Repeat("*", len(pass)))
	}
	uri := url.URL{
		Scheme: scheme,
		Host:   host,
		User:   userpass,
	}
	clean := url.URL{
		Scheme: scheme,
		Host:   host,
		User:   userwopass,
	}
	wizard.smtpStr.Text("Connection URL: " +
		strings.ReplaceAll(clean.String(), "%2A", "*"))
	wizard.smtpUrl = uri
	return uri
}

func (wizard *AccountWizard) Invalidate() {
	wizard.DoInvalidate(wizard)
}

func (wizard *AccountWizard) Draw(ctx *ui.Context) {
	wizard.steps[wizard.step].Draw(ctx)
}

func (wizard *AccountWizard) getInteractive() []ui.Interactive {
	switch wizard.step {
	case CONFIGURE_BASICS:
		return wizard.basics
	case CONFIGURE_INCOMING:
		return wizard.incoming
	case CONFIGURE_OUTGOING:
		return wizard.outgoing
	case CONFIGURE_COMPLETE:
		return wizard.complete
	}
	return nil
}

func (wizard *AccountWizard) advance(direction string) {
	wizard.Focus(false)
	if direction == "Next" && wizard.step < len(wizard.steps)-1 {
		wizard.step++
	}
	if direction == "Previous" && wizard.step > 0 {
		wizard.step--
	}
	wizard.focus = 0
	wizard.Focus(true)
	wizard.Invalidate()
}

func (wizard *AccountWizard) Focus(focus bool) {
	if interactive := wizard.getInteractive(); interactive != nil {
		interactive[wizard.focus].Focus(focus)
	}
}

func (wizard *AccountWizard) Event(event tcell.Event) bool {
	interactive := wizard.getInteractive()
	switch event := event.(type) {
	case *tcell.EventKey:
		switch event.Key() {
		case tcell.KeyUp:
			fallthrough
		case tcell.KeyBacktab:
			fallthrough
		case tcell.KeyCtrlK:
			if interactive != nil {
				interactive[wizard.focus].Focus(false)
				wizard.focus--
				if wizard.focus < 0 {
					wizard.focus = len(interactive) - 1
				}
				interactive[wizard.focus].Focus(true)
			}
			wizard.Invalidate()
			return true
		case tcell.KeyDown:
			fallthrough
		case tcell.KeyTab:
			fallthrough
		case tcell.KeyCtrlJ:
			if interactive != nil {
				interactive[wizard.focus].Focus(false)
				wizard.focus++
				if wizard.focus >= len(interactive) {
					wizard.focus = 0
				}
				interactive[wizard.focus].Focus(true)
			}
			wizard.Invalidate()
			return true
		}
	}
	if interactive != nil {
		return interactive[wizard.focus].Event(event)
	}
	return false
}

type selecter struct {
	ui.Invalidatable
	chooser bool
	focused bool
	focus   int
	options []string

	onChoose func(option string)
	onSelect func(option string)
}

func newSelecter(options []string, focus int) *selecter {
	return &selecter{
		focus:   focus,
		options: options,
	}
}

func (sel *selecter) Chooser(chooser bool) *selecter {
	sel.chooser = chooser
	return sel
}

func (sel *selecter) Invalidate() {
	sel.DoInvalidate(sel)
}

func (sel *selecter) Draw(ctx *ui.Context) {
	ctx.Fill(0, 0, ctx.Width(), ctx.Height(), ' ', tcell.StyleDefault)
	x := 2
	for i, option := range sel.options {
		style := tcell.StyleDefault
		if sel.focus == i {
			if sel.focused {
				style = style.Reverse(true)
			} else if sel.chooser {
				style = style.Bold(true)
			}
		}
		x += ctx.Printf(x, 1, style, "[%s]", option)
		x += 5
	}
}

func (sel *selecter) OnChoose(fn func(option string)) *selecter {
	sel.onChoose = fn
	return sel
}

func (sel *selecter) OnSelect(fn func(option string)) *selecter {
	sel.onSelect = fn
	return sel
}

func (sel *selecter) Selected() string {
	return sel.options[sel.focus]
}

func (sel *selecter) Focus(focus bool) {
	sel.focused = focus
	sel.Invalidate()
}

func (sel *selecter) Event(event tcell.Event) bool {
	switch event := event.(type) {
	case *tcell.EventKey:
		switch event.Key() {
		case tcell.KeyCtrlH:
			fallthrough
		case tcell.KeyLeft:
			if sel.focus > 0 {
				sel.focus--
				sel.Invalidate()
			}
			if sel.onSelect != nil {
				sel.onSelect(sel.Selected())
			}
		case tcell.KeyCtrlL:
			fallthrough
		case tcell.KeyRight:
			if sel.focus < len(sel.options)-1 {
				sel.focus++
				sel.Invalidate()
			}
			if sel.onSelect != nil {
				sel.onSelect(sel.Selected())
			}
		case tcell.KeyEnter:
			if sel.onChoose != nil {
				sel.onChoose(sel.Selected())
			}
		}
	}
	return false
}

func getSRV(host string, services []string) (string, string) {
	var hostport, srv string
	for _, srv = range services {
		_, addrs, err := net.LookupSRV(srv, "tcp", host)
		if err != nil {
			continue
		}
		if addrs[0].Target != "" && addrs[0].Port > 0 {
			hostport = net.JoinHostPort(
				strings.TrimSuffix(addrs[0].Target, "."),
				strconv.Itoa(int(addrs[0].Port)))
			break
		}
	}
	return hostport, srv
}
