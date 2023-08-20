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

	"github.com/emersion/go-message/mail"
	"github.com/gdamore/tcell/v2"
	"github.com/go-ini/ini"
	"github.com/kyoh86/xdg"
	"github.com/mitchellh/go-homedir"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib/format"
	"git.sr.ht/~rjarry/aerc/lib/ui"
	"git.sr.ht/~rjarry/aerc/log"
)

const (
	CONFIGURE_BASICS   = iota
	CONFIGURE_SOURCE   = iota
	CONFIGURE_OUTGOING = iota
	CONFIGURE_COMPLETE = iota
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
	discovered  map[string]string
	fullName    *ui.TextInput
	basics      []ui.Interactive
	// CONFIGURE_SOURCE
	sourceProtocol  *Selector
	sourceTransport *Selector

	sourceUsername *ui.TextInput
	sourcePassword *ui.TextInput
	sourceServer   *ui.TextInput
	sourceStr      *ui.Text
	sourceUrl      url.URL
	source         []ui.Interactive
	// CONFIGURE_OUTGOING
	outgoingProtocol  *Selector
	outgoingTransport *Selector

	outgoingUsername *ui.TextInput
	outgoingPassword *ui.TextInput
	outgoingServer   *ui.TextInput
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

type configStep struct {
	introduction string
	labels       []string
	fields       []ui.Drawable
	interactive  *[]ui.Interactive
}

func NewConfigStep(intro string, interactive *[]ui.Interactive) configStep {
	return configStep{introduction: intro, interactive: interactive}
}

func (s *configStep) AddField(label string, field ui.Drawable) {
	s.labels = append(s.labels, label)
	s.fields = append(s.fields, field)
	if i, ok := field.(ui.Interactive); ok {
		*s.interactive = append(*s.interactive, i)
	}
}

func (s *configStep) Grid() *ui.Grid {
	introduction := strings.TrimSpace(s.introduction)
	h := strings.Count(introduction, "\n") + 1
	spec := []ui.GridSpec{
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(1)}, // padding
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(h)}, // intro text
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(1)}, // padding
	}
	for range s.fields {
		spec = append(spec, []ui.GridSpec{
			{Strategy: ui.SIZE_EXACT, Size: ui.Const(1)}, // label
			{Strategy: ui.SIZE_EXACT, Size: ui.Const(1)}, // field
			{Strategy: ui.SIZE_EXACT, Size: ui.Const(1)}, // padding
		}...)
	}
	justify := ui.GridSpec{Strategy: ui.SIZE_WEIGHT, Size: ui.Const(1)}
	spec = append(spec, justify)
	grid := ui.NewGrid().Rows(spec).Columns([]ui.GridSpec{justify})

	intro := ui.NewText(introduction, config.Ui.GetStyle(config.STYLE_DEFAULT))
	fill := ui.NewFill(' ', tcell.StyleDefault)

	grid.AddChild(fill).At(0, 0)
	grid.AddChild(intro).At(1, 0)
	grid.AddChild(fill).At(2, 0)

	row := 3
	for i, field := range s.fields {
		label := ui.NewText(s.labels[i], config.Ui.GetStyle(config.STYLE_HEADER))
		grid.AddChild(label).At(row, 0)
		grid.AddChild(field).At(row+1, 0)
		grid.AddChild(fill).At(row+2, 0)
		row += 3
	}

	grid.AddChild(fill).At(row, 0)

	return grid
}

const (
	// protocols
	IMAP = "IMAP"
	SMTP = "SMTP"
	// transports
	SSL_TLS  = "SSL/TLS"
	OAUTH    = "SSL/TLS+OAUTHBEARER"
	XOAUTH   = "SSL/TLS+XOAUTH2"
	STARTTLS = "STARTTLS"
	INSECURE = "Insecure"
)

var (
	sources    = []string{IMAP}
	outgoings  = []string{SMTP}
	transports = []string{SSL_TLS, OAUTH, XOAUTH, STARTTLS, INSECURE}
)

func NewAccountWizard(aerc *Aerc) *AccountWizard {
	wizard := &AccountWizard{
		accountName:      ui.NewTextInput("", config.Ui).Prompt("> "),
		aerc:             aerc,
		temporary:        false,
		email:            ui.NewTextInput("", config.Ui).Prompt("> "),
		fullName:         ui.NewTextInput("", config.Ui).Prompt("> "),
		sourcePassword:   ui.NewTextInput("", config.Ui).Prompt("] ").Password(true),
		sourceServer:     ui.NewTextInput("", config.Ui).Prompt("> "),
		sourceStr:        ui.NewText("", config.Ui.GetStyle(config.STYLE_DEFAULT)),
		sourceUsername:   ui.NewTextInput("", config.Ui).Prompt("> "),
		outgoingPassword: ui.NewTextInput("", config.Ui).Prompt("] ").Password(true),
		outgoingServer:   ui.NewTextInput("", config.Ui).Prompt("> "),
		outgoingStr:      ui.NewText("", config.Ui.GetStyle(config.STYLE_DEFAULT)),
		outgoingUsername: ui.NewTextInput("", config.Ui).Prompt("> "),
		outgoingCopyTo:   ui.NewTextInput("", config.Ui).Prompt("> "),

		sourceProtocol:    NewSelector(sources, 0, config.Ui).Chooser(true),
		sourceTransport:   NewSelector(transports, 0, config.Ui).Chooser(true),
		outgoingProtocol:  NewSelector(outgoings, 0, config.Ui).Chooser(true),
		outgoingTransport: NewSelector(transports, 0, config.Ui).Chooser(true),
	}

	// Autofill some stuff for the user
	wizard.email.OnFocusLost(func(_ *ui.TextInput) {
		value := wizard.email.String()
		if wizard.sourceUsername.String() == "" {
			wizard.sourceUsername.Set(value)
		}
		if wizard.outgoingUsername.String() == "" {
			wizard.outgoingUsername.Set(value)
		}
		wizard.sourceUri()
		wizard.outgoingUri()
	})
	wizard.sourceProtocol.OnSelect(func(option string) {
		wizard.sourceServer.Set("")
		wizard.autofill()
		wizard.sourceUri()
	})
	wizard.sourceServer.OnChange(func(_ *ui.TextInput) {
		wizard.sourceUri()
	})
	wizard.sourceServer.OnFocusLost(func(_ *ui.TextInput) {
		src := wizard.sourceServer.String()
		out := wizard.outgoingServer.String()
		if out == "" && strings.HasPrefix(src, "imap.") {
			out = strings.Replace(src, "imap.", "smtp.", 1)
			wizard.outgoingServer.Set(out)
		}
		wizard.outgoingUri()
	})
	wizard.sourceUsername.OnChange(func(_ *ui.TextInput) {
		wizard.sourceUri()
	})
	wizard.sourceUsername.OnFocusLost(func(_ *ui.TextInput) {
		if wizard.outgoingUsername.String() == "" {
			wizard.outgoingUsername.Set(wizard.sourceUsername.String())
			wizard.outgoingUri()
		}
	})
	wizard.sourceTransport.OnSelect(func(option string) {
		wizard.sourceUri()
	})
	var once sync.Once
	wizard.sourcePassword.OnChange(func(_ *ui.TextInput) {
		wizard.outgoingPassword.Set(wizard.sourcePassword.String())
		wizard.sourceUri()
		wizard.outgoingUri()
	})
	wizard.sourcePassword.OnFocusLost(func(_ *ui.TextInput) {
		if wizard.sourcePassword.String() != "" {
			once.Do(func() {
				showPasswordWarning(aerc)
			})
		}
	})
	wizard.outgoingProtocol.OnSelect(func(option string) {
		wizard.outgoingServer.Set("")
		wizard.autofill()
		wizard.outgoingUri()
	})
	wizard.outgoingServer.OnChange(func(_ *ui.TextInput) {
		wizard.outgoingUri()
	})
	wizard.outgoingUsername.OnChange(func(_ *ui.TextInput) {
		wizard.outgoingUri()
	})
	wizard.outgoingPassword.OnChange(func(_ *ui.TextInput) {
		if wizard.outgoingPassword.String() != "" {
			once.Do(func() {
				showPasswordWarning(aerc)
			})
		}
		wizard.outgoingUri()
	})
	wizard.outgoingTransport.OnSelect(func(option string) {
		wizard.outgoingUri()
	})

	// CONFIGURE_BASICS
	basics := NewConfigStep(
		`
Welcome to aerc! Let's configure your account.

This wizard supports basic IMAP & SMTP configuration.
For other configurations, use <Ctrl+q> to exit and read the aerc-accounts(5) man page.
Press <Tab> and <Shift+Tab> to cycle between each field in this form, or <Ctrl+j> and <Ctrl+k>.
`,
		&wizard.basics,
	)
	basics.AddField(
		"Name for this account? (e.g. 'Personal' or 'Work')",
		wizard.accountName,
	)
	basics.AddField(
		"Full name for outgoing emails? (e.g. 'John Doe')",
		wizard.fullName,
	)
	basics.AddField(
		"Your email address? (e.g. 'john@example.org')",
		wizard.email,
	)
	basics.AddField("", NewSelector([]string{"Next"}, 0, config.Ui).
		OnChoose(func(option string) {
			wizard.discoverServices()
			wizard.autofill()
			wizard.sourceUri()
			wizard.outgoingUri()
			wizard.advance(option)
		}),
	)

	// CONFIGURE_SOURCE
	source := NewConfigStep("Configure email source", &wizard.source)
	source.AddField("Protocol", wizard.sourceProtocol)
	source.AddField("Username", wizard.sourceUsername)
	source.AddField("Password", wizard.sourcePassword)
	source.AddField(
		"Server address (e.g. 'mail.example.org' or 'mail.example.org:1313')",
		wizard.sourceServer,
	)
	source.AddField("Transport security", wizard.sourceTransport)
	source.AddField("Connection URL", wizard.sourceStr)
	source.AddField(
		"", NewSelector([]string{"Previous", "Next"}, 1, config.Ui).
			OnChoose(wizard.advance),
	)

	// CONFIGURE_OUTGOING
	outgoing := NewConfigStep("Configure outgoing mail", &wizard.outgoing)
	outgoing.AddField("Protocol", wizard.outgoingProtocol)
	outgoing.AddField("Username", wizard.outgoingUsername)
	outgoing.AddField("Password", wizard.outgoingPassword)
	outgoing.AddField(
		"Server address (e.g. 'mail.example.org' or 'mail.example.org:1313')",
		wizard.outgoingServer,
	)
	outgoing.AddField("Transport security", wizard.outgoingTransport)
	outgoing.AddField("Connection URL", wizard.outgoingStr)
	outgoing.AddField(
		"Copy sent messages to folder (leave empty to disable)",
		wizard.outgoingCopyTo,
	)
	outgoing.AddField(
		"", NewSelector([]string{"Previous", "Next"}, 1, config.Ui).
			OnChoose(wizard.advance),
	)

	// CONFIGURE_COMPLETE
	complete := NewConfigStep(
		fmt.Sprintf(`
Configuration complete!

You can go back and double check your settings, or choose [Finish] to
save your settings to %s/accounts.conf.

Make sure to review the contents of this file and read the
aerc-accounts(5) man page for guidance and further tweaking.

To add another account in the future, run ':new-account'.
`, tildeHome(path.Join(xdg.ConfigHome(), "aerc"))),
		&wizard.complete,
	)
	complete.AddField(
		"", NewSelector([]string{
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
		}),
	)

	wizard.steps = []*ui.Grid{
		basics.Grid(), source.Grid(), outgoing.Grid(), complete.Grid(),
	}

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
	from := mail.Address{
		Name:    wizard.fullName.String(),
		Address: wizard.email.String(),
	}
	_, _ = sec.NewKey("from", format.AddressForHumans(&from))
	if wizard.outgoingCopyTo.String() != "" {
		_, _ = sec.NewKey("copy-to", wizard.outgoingCopyTo.String())
	}

	if wizard.sourceProtocol.Selected() == IMAP {
		_, _ = sec.NewKey("cache-headers", "true")
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

func splitHostPath(server string) (string, string) {
	host, path, found := strings.Cut(server, "/")
	if found {
		path = "/" + path
	}
	return host, path
}

func makeURLs(scheme, host, path, user, pass string) (url.URL, url.URL) {
	var opaque string

	// If everything is unset, the rendered URL is '<scheme>:'.
	// Force a '//' opaque suffix so that it is rendered as '<scheme>://'.
	if scheme != "" && host == "" && path == "" && user == "" && pass == "" {
		opaque = "//"
	}

	uri := url.URL{Scheme: scheme, Host: host, Path: path, Opaque: opaque}
	clean := uri

	switch {
	case pass != "":
		uri.User = url.UserPassword(user, pass)
		clean.User = url.UserPassword(user, strings.Repeat("*", len(pass)))
	case user != "":
		uri.User = url.User(user)
		clean.User = url.User(user)
	}

	return uri, clean
}

func (wizard *AccountWizard) sourceUri() url.URL {
	host, path := splitHostPath(wizard.sourceServer.String())
	user := wizard.sourceUsername.String()
	pass := wizard.sourcePassword.String()
	var scheme string
	if wizard.sourceProtocol.Selected() == IMAP {
		switch wizard.sourceTransport.Selected() {
		case STARTTLS:
			scheme = "imap"
		case INSECURE:
			scheme = "imap+insecure"
		case OAUTH:
			scheme = "imaps+oauthbearer"
		case XOAUTH:
			scheme = "imaps+xoauth2"
		default:
			scheme = "imaps"
		}
	}

	uri, clean := makeURLs(scheme, host, path, user, pass)

	wizard.sourceStr.Text(
		"  " + strings.ReplaceAll(clean.String(), "%2A", "*"))
	wizard.sourceUrl = uri
	return uri
}

func (wizard *AccountWizard) outgoingUri() url.URL {
	host, path := splitHostPath(wizard.outgoingServer.String())
	user := wizard.outgoingUsername.String()
	pass := wizard.outgoingPassword.String()
	var scheme string
	if wizard.outgoingProtocol.Selected() == SMTP {
		switch wizard.outgoingTransport.Selected() {
		case OAUTH:
			scheme = "smtps+oauthbearer"
		case XOAUTH:
			scheme = "smtps+xoauth2"
		case INSECURE:
			scheme = "smtp+insecure"
		case STARTTLS:
			scheme = "smtp"
		default:
			scheme = "smtps"
		}
	}

	uri, clean := makeURLs(scheme, host, path, user, pass)

	wizard.outgoingStr.Text(
		"  " + strings.ReplaceAll(clean.String(), "%2A", "*"))
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

func (wizard *AccountWizard) discoverServices() {
	email := wizard.email.String()
	if !strings.ContainsRune(email, '@') {
		return
	}
	domain := email[strings.IndexRune(email, '@')+1:]
	var wg sync.WaitGroup
	type Service struct{ srv, hostport string }
	services := make(chan Service)

	for _, service := range []string{"imaps", "imap", "submission"} {
		wg.Add(1)
		go func(srv string) {
			defer log.PanicHandler()
			defer wg.Done()
			_, addrs, err := net.LookupSRV(srv, "tcp", domain)
			if err != nil {
				log.Tracef("SRV lookup for _%s._tcp.%s failed: %s",
					srv, domain, err)
			} else if addrs[0].Target != "" && addrs[0].Port > 0 {
				services <- Service{
					srv: srv,
					hostport: net.JoinHostPort(
						strings.TrimSuffix(addrs[0].Target, "."),
						strconv.Itoa(int(addrs[0].Port))),
				}
			}
		}(service)
	}
	go func() {
		defer log.PanicHandler()
		wg.Wait()
		close(services)
	}()

	wizard.discovered = make(map[string]string)
	for s := range services {
		wizard.discovered[s.srv] = s.hostport
	}
}

func (wizard *AccountWizard) autofill() {
	if wizard.sourceServer.String() == "" {
		if wizard.sourceProtocol.Selected() == IMAP {
			if s, ok := wizard.discovered["imaps"]; ok {
				wizard.sourceServer.Set(s)
				wizard.sourceTransport.Select(SSL_TLS)
			} else if s, ok := wizard.discovered["imap"]; ok {
				wizard.sourceServer.Set(s)
				wizard.sourceTransport.Select(STARTTLS)
			}
		}
	}
	if wizard.outgoingServer.String() == "" {
		if wizard.outgoingProtocol.Selected() == SMTP {
			if s, ok := wizard.discovered["submission"]; ok {
				switch {
				case strings.HasSuffix(s, ":587"):
					wizard.outgoingTransport.Select(SSL_TLS)
				case strings.HasSuffix(s, ":465"):
					wizard.outgoingTransport.Select(STARTTLS)
				default:
					wizard.outgoingTransport.Select(INSECURE)
				}
				wizard.outgoingServer.Set(s)
			}
		}
	}
}

func tildeHome(path string) string {
	home, err := homedir.Dir()
	if err == nil && home != "" && strings.HasPrefix(path, home) {
		path = "~" + strings.TrimPrefix(path, home)
	}
	return path
}
