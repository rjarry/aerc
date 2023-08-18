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
	CONFIGURE_SOURCE   = iota
	CONFIGURE_OUTGOING = iota
	CONFIGURE_COMPLETE = iota
)

const (
	SSL_TLS  = iota
	STARTTLS = iota
	INSECURE = iota
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
	// CONFIGURE_SOURCE
	sourceUsername *ui.TextInput
	sourcePassword *ui.TextInput
	sourceServer   *ui.TextInput
	sourceMode     int
	sourceStr      *ui.Text
	sourceUrl      url.URL
	source         []ui.Interactive
	// CONFIGURE_OUTGOING
	outgoingUsername *ui.TextInput
	outgoingPassword *ui.TextInput
	outgoingServer   *ui.TextInput
	outgoingMode     int
	outgoingStr      *ui.Text
	outgoingUrl      url.URL
	outgoingCopyTo   *ui.TextInput
	outgoing         []ui.Interactive
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
		accountName:      ui.NewTextInput("", config.Ui).Prompt("> "),
		aerc:             aerc,
		temporary:        false,
		email:            ui.NewTextInput("", config.Ui).Prompt("> "),
		fullName:         ui.NewTextInput("", config.Ui).Prompt("> "),
		sourcePassword:   ui.NewTextInput("", config.Ui).Prompt("] ").Password(true),
		sourceServer:     ui.NewTextInput("", config.Ui).Prompt("> "),
		sourceStr:        ui.NewText("Connection URL: imaps://", config.Ui.GetStyle(config.STYLE_DEFAULT)),
		sourceUsername:   ui.NewTextInput("", config.Ui).Prompt("> "),
		outgoingPassword: ui.NewTextInput("", config.Ui).Prompt("] ").Password(true),
		outgoingServer:   ui.NewTextInput("", config.Ui).Prompt("> "),
		outgoingStr:      ui.NewText("Connection URL: smtps://", config.Ui.GetStyle(config.STYLE_DEFAULT)),
		outgoingUsername: ui.NewTextInput("", config.Ui).Prompt("> "),
		outgoingCopyTo:   ui.NewTextInput("", config.Ui).Prompt("> "),
	}

	// Autofill some stuff for the user
	wizard.email.OnChange(func(_ *ui.TextInput) {
		value := wizard.email.String()
		wizard.sourceUsername.Set(value)
		wizard.outgoingUsername.Set(value)
		if strings.ContainsRune(value, '@') {
			server := value[strings.IndexRune(value, '@')+1:]
			wizard.sourceServer.Set(server)
			wizard.outgoingServer.Set(server)
		}
		wizard.sourceUri()
		wizard.outgoingUri()
	})
	wizard.sourceServer.OnChange(func(_ *ui.TextInput) {
		sourceServerURI := wizard.sourceServer.String()
		outgoingServerURI := sourceServerURI
		if strings.HasPrefix(sourceServerURI, "imap.") {
			outgoingServerURI = strings.Replace(sourceServerURI, "imap.", "smtp.", 1)
		}
		wizard.outgoingServer.Set(outgoingServerURI)
		wizard.sourceUri()
		wizard.outgoingUri()
	})
	wizard.sourceUsername.OnChange(func(_ *ui.TextInput) {
		wizard.outgoingUsername.Set(wizard.sourceUsername.String())
		wizard.sourceUri()
		wizard.outgoingUri()
	})
	var once sync.Once
	wizard.sourcePassword.OnChange(func(_ *ui.TextInput) {
		wizard.outgoingPassword.Set(wizard.sourcePassword.String())
		wizard.sourceUri()
		wizard.outgoingUri()
	})
	wizard.sourcePassword.OnFocusLost(func(_ *ui.TextInput) {
		once.Do(func() {
			showPasswordWarning(aerc)
		})
	})
	wizard.outgoingServer.OnChange(func(_ *ui.TextInput) {
		wizard.outgoingUri()
	})
	wizard.outgoingUsername.OnChange(func(_ *ui.TextInput) {
		wizard.outgoingUri()
	})
	wizard.outgoingPassword.OnChange(func(_ *ui.TextInput) {
		wizard.outgoingUri()
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
					wizard.sourceServer.Set(hostport)
					if srv == "imaps" {
						wizard.sourceMode = SSL_TLS
					} else {
						wizard.sourceMode = STARTTLS
					}
					wizard.sourceUri()
				}
				hostport, _ = getSRV(server, []string{"submission"})
				if hostport != "" {
					wizard.outgoingServer.Set(hostport)
					wizard.outgoingMode = STARTTLS
					wizard.outgoingUri()
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
	incoming.AddChild(wizard.sourceUsername).
		At(2, 0)
	incoming.AddChild(ui.NewFill(' ', tcell.StyleDefault)).
		At(3, 0)
	incoming.AddChild(
		ui.NewText("Password",
			config.Ui.GetStyle(config.STYLE_HEADER))).
		At(4, 0)
	incoming.AddChild(wizard.sourcePassword).
		At(5, 0)
	incoming.AddChild(ui.NewFill(' ', tcell.StyleDefault)).
		At(6, 0)
	incoming.AddChild(
		ui.NewText("Server address "+
			"(e.g. 'mail.example.org' or 'mail.example.org:1313')",
			config.Ui.GetStyle(config.STYLE_HEADER))).
		At(7, 0)
	incoming.AddChild(wizard.sourceServer).
		At(8, 0)
	incoming.AddChild(ui.NewFill(' ', tcell.StyleDefault)).
		At(9, 0)
	incoming.AddChild(
		ui.NewText("Connection mode",
			config.Ui.GetStyle(config.STYLE_HEADER))).
		At(10, 0)
	sourceMode := NewSelector([]string{
		"IMAP over SSL/TLS",
		"IMAP with STARTTLS",
		"Insecure IMAP",
	}, 0, config.Ui).Chooser(true).OnSelect(func(option string) {
		switch option {
		case "IMAP over SSL/TLS":
			wizard.sourceMode = SSL_TLS
		case "IMAP with STARTTLS":
			wizard.sourceMode = STARTTLS
		case "Insecure IMAP":
			wizard.sourceMode = INSECURE
		}
		wizard.sourceUri()
	})
	incoming.AddChild(sourceMode).At(11, 0)
	selector = NewSelector([]string{"Previous", "Next"}, 1, config.Ui).
		OnChoose(wizard.advance)
	incoming.AddChild(ui.NewFill(' ', tcell.StyleDefault)).At(12, 0)
	incoming.AddChild(wizard.sourceStr).At(13, 0)
	incoming.AddChild(selector).At(14, 0)
	wizard.source = []ui.Interactive{
		wizard.sourceUsername,
		wizard.sourcePassword,
		wizard.sourceServer,
		sourceMode, selector,
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
	outgoing.AddChild(wizard.outgoingUsername).
		At(2, 0)
	outgoing.AddChild(ui.NewFill(' ', tcell.StyleDefault)).
		At(3, 0)
	outgoing.AddChild(
		ui.NewText("Password",
			config.Ui.GetStyle(config.STYLE_HEADER))).
		At(4, 0)
	outgoing.AddChild(wizard.outgoingPassword).
		At(5, 0)
	outgoing.AddChild(ui.NewFill(' ', tcell.StyleDefault)).
		At(6, 0)
	outgoing.AddChild(
		ui.NewText("Server address "+
			"(e.g. 'mail.example.org' or 'mail.example.org:1313')",
			config.Ui.GetStyle(config.STYLE_HEADER))).
		At(7, 0)
	outgoing.AddChild(wizard.outgoingServer).
		At(8, 0)
	outgoing.AddChild(ui.NewFill(' ', tcell.StyleDefault)).
		At(9, 0)
	outgoing.AddChild(
		ui.NewText("Connection mode",
			config.Ui.GetStyle(config.STYLE_HEADER))).
		At(10, 0)
	outgoingMode := NewSelector([]string{
		"SMTP over SSL/TLS",
		"SMTP with STARTTLS",
		"Insecure SMTP",
	}, 0, config.Ui).Chooser(true).OnSelect(func(option string) {
		switch option {
		case "SMTP over SSL/TLS":
			wizard.outgoingMode = SSL_TLS
		case "SMTP with STARTTLS":
			wizard.outgoingMode = STARTTLS
		case "Insecure SMTP":
			wizard.outgoingMode = INSECURE
		}
		wizard.outgoingUri()
	})
	outgoing.AddChild(outgoingMode).At(11, 0)
	selector = NewSelector([]string{"Previous", "Next"}, 1, config.Ui).
		OnChoose(wizard.advance)
	outgoing.AddChild(ui.NewFill(' ', tcell.StyleDefault)).At(12, 0)
	outgoing.AddChild(wizard.outgoingStr).At(13, 0)
	outgoing.AddChild(ui.NewFill(' ', tcell.StyleDefault)).At(14, 0)
	outgoing.AddChild(
		ui.NewText("Copy sent messages to folder (leave empty to disable)",
			config.Ui.GetStyle(config.STYLE_HEADER))).At(15, 0)
	outgoing.AddChild(wizard.outgoingCopyTo).At(16, 0)
	outgoing.AddChild(selector).At(17, 0)
	wizard.outgoing = []ui.Interactive{
		wizard.outgoingUsername,
		wizard.outgoingPassword,
		wizard.outgoingServer,
		outgoingMode, wizard.outgoingCopyTo, selector,
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
		wizard.source,
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
	if wizard.sourceServer.String() == "" {
		wizard.errorFor(wizard.sourceServer,
			errors.New("Email source configuration is required"))
		return
	}
	if wizard.outgoingServer.String() == "" {
		wizard.errorFor(wizard.outgoingServer,
			errors.New("Outgoing mail configuration is required"))
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
	// these can't fail
	_, _ = sec.NewKey("source", wizard.sourceUrl.String())
	_, _ = sec.NewKey("outgoing", wizard.outgoingUrl.String())
	_, _ = sec.NewKey("default", "INBOX")
	_, _ = sec.NewKey("from", fmt.Sprintf("%s <%s>",
		wizard.fullName.String(), wizard.email.String()))
	if wizard.outgoingCopyTo.String() != "" {
		_, _ = sec.NewKey("copy-to", wizard.outgoingCopyTo.String())
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

func (wizard *AccountWizard) sourceUri() url.URL {
	host := wizard.sourceServer.String()
	user := wizard.sourceUsername.String()
	pass := wizard.sourcePassword.String()
	var scheme string
	switch wizard.sourceMode {
	case SSL_TLS:
		scheme = "imaps"
	case STARTTLS:
		scheme = "imap"
	case INSECURE:
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
	wizard.sourceStr.Text("Connection URL: " +
		strings.ReplaceAll(clean.String(), "%2A", "*"))
	wizard.sourceUrl = uri
	return uri
}

func (wizard *AccountWizard) outgoingUri() url.URL {
	host := wizard.outgoingServer.String()
	user := wizard.outgoingUsername.String()
	pass := wizard.outgoingPassword.String()
	var scheme string
	switch wizard.outgoingMode {
	case SSL_TLS:
		scheme = "smtps"
	case STARTTLS:
		scheme = "smtp"
	case INSECURE:
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
	wizard.outgoingStr.Text("Connection URL: " +
		strings.ReplaceAll(clean.String(), "%2A", "*"))
	wizard.outgoingUrl = uri
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
	case CONFIGURE_SOURCE:
		return wizard.source
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
