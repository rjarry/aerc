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
	"sync"

	"github.com/gdamore/tcell/v2"
	"github.com/go-ini/ini"
	"github.com/kyoh86/xdg"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib/ui"
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
	aerc      *Aerc
	step      int
	steps     []*ui.Grid
	focus     int
	temporary bool
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

func showPasswordWarning(aerc *Aerc) {
	title := "ATTENTION"
	text := `
The Wizard will store your passwords as clear text in:

  ~/.config/aerc/accounts.conf

It is recommended to remove the clear text passwords and configure
'source-cred-cmd' and 'outgoing-cred-cmd' using your own password store
after the setup.
`
	warning := NewSelectorDialog(
		title, text, []string{"OK"}, 0,
		aerc.SelectedAccountUiConfig(),
		func(_ string, _ error) {
			aerc.CloseDialog()
		},
	)
	aerc.AddDialog(warning)
}

func NewAccountWizard(aerc *Aerc) *AccountWizard {
	wizard := &AccountWizard{
		accountName:  ui.NewTextInput("", config.Ui).Prompt("> "),
		aerc:         aerc,
		temporary:    false,
		copySent:     true,
		email:        ui.NewTextInput("", config.Ui).Prompt("> "),
		fullName:     ui.NewTextInput("", config.Ui).Prompt("> "),
		imapPassword: ui.NewTextInput("", config.Ui).Prompt("] ").Password(true),
		imapServer:   ui.NewTextInput("", config.Ui).Prompt("> "),
		imapStr:      ui.NewText("Connection URL: imaps://", config.Ui.GetStyle(config.STYLE_DEFAULT)),
		imapUsername: ui.NewTextInput("", config.Ui).Prompt("> "),
		smtpPassword: ui.NewTextInput("", config.Ui).Prompt("] ").Password(true),
		smtpServer:   ui.NewTextInput("", config.Ui).Prompt("> "),
		smtpStr:      ui.NewText("Connection URL: smtps://", config.Ui.GetStyle(config.STYLE_DEFAULT)),
		smtpUsername: ui.NewTextInput("", config.Ui).Prompt("> "),
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
	var once sync.Once
	wizard.imapPassword.OnChange(func(_ *ui.TextInput) {
		wizard.smtpPassword.Set(wizard.imapPassword.String())
		wizard.imapUri()
		wizard.smtpUri()
	})
	wizard.imapPassword.OnFocusLost(func(_ *ui.TextInput) {
		once.Do(func() {
			showPasswordWarning(aerc)
		})
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
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(8)}, // Introduction
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(1)}, // Account name (label)
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(1)}, // (input)
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(1)}, // Padding
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(1)}, // Full name (label)
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(1)}, // (input)
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(1)}, // Padding
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(1)}, // Email address (label)
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(1)}, // (input)
		{Strategy: ui.SIZE_WEIGHT, Size: ui.Const(1)},
	}).Columns([]ui.GridSpec{
		{Strategy: ui.SIZE_WEIGHT, Size: ui.Const(1)},
	})
	basics.AddChild(
		ui.NewText("\nWelcome to aerc! Let's configure your account.\n\n"+
			"This wizard supports basic IMAP & SMTP configuration.\n"+
			"For other configurations, use <Ctrl+q> to exit and read the "+
			"aerc-accounts(5) man page.\n"+
			"Press <Tab> and <Shift+Tab> to cycle between each field in this form, "+
			"or <Ctrl+j> and <Ctrl+k>.",
			config.Ui.GetStyle(config.STYLE_DEFAULT)))
	basics.AddChild(
		ui.NewText("Name for this account? (e.g. 'Personal' or 'Work')",
			config.Ui.GetStyle(config.STYLE_HEADER))).
		At(1, 0)
	basics.AddChild(wizard.accountName).
		At(2, 0)
	basics.AddChild(ui.NewFill(' ', tcell.StyleDefault)).
		At(3, 0)
	basics.AddChild(
		ui.NewText("Full name for outgoing emails? (e.g. 'John Doe')",
			config.Ui.GetStyle(config.STYLE_HEADER))).
		At(4, 0)
	basics.AddChild(wizard.fullName).
		At(5, 0)
	basics.AddChild(ui.NewFill(' ', tcell.StyleDefault)).
		At(6, 0)
	basics.AddChild(
		ui.NewText("Your email address? (e.g. 'john@example.org')",
			config.Ui.GetStyle(config.STYLE_HEADER))).
		At(7, 0)
	basics.AddChild(wizard.email).
		At(8, 0)
	selector := NewSelector([]string{"Next"}, 0, config.Ui).
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
				hostport, _ = getSRV(server, []string{"submission"})
				if hostport != "" {
					wizard.smtpServer.Set(hostport)
					wizard.smtpMode = SMTP_STARTTLS
					wizard.smtpUri()
				}
			}
			wizard.advance(option)
		})
	basics.AddChild(selector).At(9, 0)
	wizard.basics = []ui.Interactive{
		wizard.accountName, wizard.fullName, wizard.email, selector,
	}

	incoming := ui.NewGrid().Rows([]ui.GridSpec{
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(3)}, // Introduction
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(1)}, // Username (label)
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(1)}, // (input)
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(1)}, // Padding
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(1)}, // Password (label)
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(1)}, // (input)
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(1)}, // Padding
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(1)}, // Server (label)
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(1)}, // (input)
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(1)}, // Padding
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(1)}, // Connection mode (label)
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(2)}, // (input)
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(1)}, // Padding
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(1)}, // Connection string
		{Strategy: ui.SIZE_WEIGHT, Size: ui.Const(1)},
	}).Columns([]ui.GridSpec{
		{Strategy: ui.SIZE_WEIGHT, Size: ui.Const(1)},
	})
	incoming.AddChild(ui.NewText("\nConfigure incoming mail (IMAP)",
		config.Ui.GetStyle(config.STYLE_DEFAULT)))
	incoming.AddChild(
		ui.NewText("Username",
			config.Ui.GetStyle(config.STYLE_HEADER))).
		At(1, 0)
	incoming.AddChild(wizard.imapUsername).
		At(2, 0)
	incoming.AddChild(ui.NewFill(' ', tcell.StyleDefault)).
		At(3, 0)
	incoming.AddChild(
		ui.NewText("Password",
			config.Ui.GetStyle(config.STYLE_HEADER))).
		At(4, 0)
	incoming.AddChild(wizard.imapPassword).
		At(5, 0)
	incoming.AddChild(ui.NewFill(' ', tcell.StyleDefault)).
		At(6, 0)
	incoming.AddChild(
		ui.NewText("Server address "+
			"(e.g. 'mail.example.org' or 'mail.example.org:1313')",
			config.Ui.GetStyle(config.STYLE_HEADER))).
		At(7, 0)
	incoming.AddChild(wizard.imapServer).
		At(8, 0)
	incoming.AddChild(ui.NewFill(' ', tcell.StyleDefault)).
		At(9, 0)
	incoming.AddChild(
		ui.NewText("Connection mode",
			config.Ui.GetStyle(config.STYLE_HEADER))).
		At(10, 0)
	imapMode := NewSelector([]string{
		"IMAP over SSL/TLS",
		"IMAP with STARTTLS",
		"Insecure IMAP",
	}, 0, config.Ui).Chooser(true).OnSelect(func(option string) {
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
	selector = NewSelector([]string{"Previous", "Next"}, 1, config.Ui).
		OnChoose(wizard.advance)
	incoming.AddChild(ui.NewFill(' ', tcell.StyleDefault)).At(12, 0)
	incoming.AddChild(wizard.imapStr).At(13, 0)
	incoming.AddChild(selector).At(14, 0)
	wizard.incoming = []ui.Interactive{
		wizard.imapUsername, wizard.imapPassword, wizard.imapServer,
		imapMode, selector,
	}

	outgoing := ui.NewGrid().Rows([]ui.GridSpec{
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(3)}, // Introduction
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(1)}, // Username (label)
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(1)}, // (input)
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(1)}, // Padding
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(1)}, // Password (label)
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(1)}, // (input)
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(1)}, // Padding
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(1)}, // Server (label)
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(1)}, // (input)
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(1)}, // Padding
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(1)}, // Connection mode (label)
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(2)}, // (input)
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(1)}, // Padding
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(1)}, // Connection string
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(1)}, // Padding
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(1)}, // Copy to sent (label)
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(2)}, // (input)
		{Strategy: ui.SIZE_WEIGHT, Size: ui.Const(1)},
	}).Columns([]ui.GridSpec{
		{Strategy: ui.SIZE_WEIGHT, Size: ui.Const(1)},
	})
	outgoing.AddChild(ui.NewText("\nConfigure outgoing mail (SMTP)",
		config.Ui.GetStyle(config.STYLE_DEFAULT)))
	outgoing.AddChild(
		ui.NewText("Username",
			config.Ui.GetStyle(config.STYLE_HEADER))).
		At(1, 0)
	outgoing.AddChild(wizard.smtpUsername).
		At(2, 0)
	outgoing.AddChild(ui.NewFill(' ', tcell.StyleDefault)).
		At(3, 0)
	outgoing.AddChild(
		ui.NewText("Password",
			config.Ui.GetStyle(config.STYLE_HEADER))).
		At(4, 0)
	outgoing.AddChild(wizard.smtpPassword).
		At(5, 0)
	outgoing.AddChild(ui.NewFill(' ', tcell.StyleDefault)).
		At(6, 0)
	outgoing.AddChild(
		ui.NewText("Server address "+
			"(e.g. 'mail.example.org' or 'mail.example.org:1313')",
			config.Ui.GetStyle(config.STYLE_HEADER))).
		At(7, 0)
	outgoing.AddChild(wizard.smtpServer).
		At(8, 0)
	outgoing.AddChild(ui.NewFill(' ', tcell.StyleDefault)).
		At(9, 0)
	outgoing.AddChild(
		ui.NewText("Connection mode",
			config.Ui.GetStyle(config.STYLE_HEADER))).
		At(10, 0)
	smtpMode := NewSelector([]string{
		"SMTP over SSL/TLS",
		"SMTP with STARTTLS",
		"Insecure SMTP",
	}, 0, config.Ui).Chooser(true).OnSelect(func(option string) {
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
	selector = NewSelector([]string{"Previous", "Next"}, 1, config.Ui).
		OnChoose(wizard.advance)
	outgoing.AddChild(ui.NewFill(' ', tcell.StyleDefault)).At(12, 0)
	outgoing.AddChild(wizard.smtpStr).At(13, 0)
	outgoing.AddChild(ui.NewFill(' ', tcell.StyleDefault)).At(14, 0)
	outgoing.AddChild(
		ui.NewText("Copy sent messages to 'Sent' folder?",
			config.Ui.GetStyle(config.STYLE_HEADER))).At(15, 0)
	copySent := NewSelector([]string{"Yes", "No"}, 0, config.Ui).
		Chooser(true).OnChoose(func(option string) {
		switch option {
		case "Yes":
			wizard.copySent = true
		case "No":
			wizard.copySent = false
		}
	})
	outgoing.AddChild(copySent).At(16, 0)
	outgoing.AddChild(selector).At(17, 0)
	wizard.outgoing = []ui.Interactive{
		wizard.smtpUsername, wizard.smtpPassword, wizard.smtpServer,
		smtpMode, copySent, selector,
	}

	complete := ui.NewGrid().Rows([]ui.GridSpec{
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(7)},  // Introduction
		{Strategy: ui.SIZE_WEIGHT, Size: ui.Const(1)}, // Previous / Finish / Finish & open tutorial
	}).Columns([]ui.GridSpec{
		{Strategy: ui.SIZE_WEIGHT, Size: ui.Const(1)},
	})
	complete.AddChild(ui.NewText(
		"\nConfiguration complete!\n\n"+
			"You can go back and double check your settings, or choose 'Finish' to\n"+
			"save your settings to accounts.conf.\n\n"+
			"To add another account in the future, run ':new-account'.",
		config.Ui.GetStyle(config.STYLE_DEFAULT)))
	selector = NewSelector([]string{
		"Previous",
		"Finish & open tutorial",
		"Finish",
	}, 1, config.Ui).OnChoose(func(option string) {
		switch option {
		case "Previous":
			wizard.advance("Previous")
		case "Finish & open tutorial":
			wizard.finish(true)
		case "Finish":
			wizard.finish(false)
		}
	})
	complete.AddChild(selector).At(1, 0)
	wizard.complete = []ui.Interactive{selector}

	wizard.steps = []*ui.Grid{basics, incoming, outgoing, complete}
	return wizard
}

func (wizard *AccountWizard) ConfigureTemporaryAccount(temporary bool) {
	wizard.temporary = temporary
}

func (wizard *AccountWizard) errorFor(d ui.Interactive, err error) {
	if d == nil {
		wizard.aerc.PushError(err.Error())
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
				wizard.aerc.PushError(err.Error())
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
	// these can't really fail
	sec.NewKey("source", wizard.imapUrl.String())   //nolint:errcheck // can't fail. option shadowing is not enabled and the key is not empty
	sec.NewKey("outgoing", wizard.smtpUrl.String()) //nolint:errcheck // can't fail. option shadowing is not enabled and the key is not empty
	sec.NewKey("default", "INBOX")                  //nolint:errcheck // can't fail. option shadowing is not enabled and the key is not empty
	sec.NewKey("from", fmt.Sprintf("%s <%s>",       //nolint:errcheck // can't fail. option shadowing is not enabled and the key is not empty
		wizard.fullName.String(), wizard.email.String()))
	if wizard.copySent {
		sec.NewKey("copy-to", "Sent") //nolint:errcheck // can't fail. option shadowing is not enabled and the key is not empty
	}

	if !wizard.temporary {
		f, err := os.OpenFile(accountsConf, os.O_WRONLY|os.O_CREATE, 0o600)
		if err != nil {
			wizard.errorFor(nil, err)
			return
		}
		defer f.Close()
		if _, err = file.WriteTo(f); err != nil {
			wizard.errorFor(nil, err)
			return
		}
	}

	account, err := config.ParseAccountConfig(sec.Name(), sec)
	if err != nil {
		wizard.errorFor(nil, err)
		return
	}
	config.Accounts = append(config.Accounts, account)

	view, err := NewAccountView(wizard.aerc, account, wizard.aerc, nil)
	if err != nil {
		wizard.aerc.NewTab(errorScreen(err.Error()), account.Name)
		return
	}
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
			wizard.aerc.RemoveTab(term, false)
			if err != nil {
				wizard.aerc.PushError(err.Error())
			}
		}
	}

	wizard.aerc.RemoveTab(wizard, false)
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
		scheme = "smtps"
	case SMTP_STARTTLS:
		scheme = "smtp"
	case SMTP_INSECURE:
		scheme = "smtp+insecure"
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
	ui.Invalidate()
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
	if event, ok := event.(*tcell.EventKey); ok {
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
