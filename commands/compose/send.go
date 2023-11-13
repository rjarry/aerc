package compose

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net/url"
	"os/exec"
	"strings"
	"time"

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
	"github.com/pkg/errors"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/commands/mode"
	"git.sr.ht/~rjarry/aerc/commands/msg"
	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/log"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
	"git.sr.ht/~rjarry/go-opt"
	"github.com/emersion/go-message/mail"
	"golang.org/x/oauth2"
)

type Send struct {
	Archive string `opt:"-a" action:"ParseArchive" metavar:"flat|year|month" complete:"CompleteArchive"`
	CopyTo  string `opt:"-t" complete:"CompleteFolders"`
}

func init() {
	register(Send{})
}

func (Send) Aliases() []string {
	return []string{"send"}
}

func (*Send) CompleteArchive(arg string) []string {
	return commands.FilterList(msg.ARCHIVE_TYPES, arg, nil)
}

func (*Send) CompleteFolders(arg string) []string {
	return commands.GetFolders(arg)
}

func (s *Send) ParseArchive(arg string) error {
	for _, a := range msg.ARCHIVE_TYPES {
		if a == arg {
			s.Archive = arg
			return nil
		}
	}
	return errors.New("unsupported archive type")
}

func (s Send) Execute(args []string) error {
	tab := app.SelectedTab()
	if tab == nil {
		return errors.New("No selected tab")
	}
	composer, _ := tab.Content.(*app.Composer)
	tabName := tab.Name

	config := composer.Config()

	if s.CopyTo == "" {
		s.CopyTo = config.CopyTo
	}

	outgoing, err := config.Outgoing.ConnectionString()
	if err != nil {
		return errors.Wrap(err, "ReadCredentials(outgoing)")
	}
	if outgoing == "" {
		return errors.New(
			"No outgoing mail transport configured for this account")
	}

	header, err := composer.PrepareHeader()
	if err != nil {
		return errors.Wrap(err, "PrepareHeader")
	}
	rcpts, err := listRecipients(header)
	if err != nil {
		return errors.Wrap(err, "listRecipients")
	}
	if len(rcpts) == 0 {
		return errors.New("Cannot send message with no recipients")
	}

	uri, err := url.Parse(outgoing)
	if err != nil {
		return errors.Wrap(err, "url.Parse(outgoing)")
	}

	scheme, auth, err := parseScheme(uri)
	if err != nil {
		return err
	}
	var domain string
	if domain_, ok := config.Params["smtp-domain"]; ok {
		domain = domain_
	}
	ctx := sendCtx{
		uri:     uri,
		scheme:  scheme,
		auth:    auth,
		from:    config.From,
		rcpts:   rcpts,
		domain:  domain,
		archive: s.Archive,
		copyto:  s.CopyTo,
	}

	log.Debugf("send config uri: %s", ctx.uri)
	log.Debugf("send config scheme: %s", ctx.scheme)
	log.Debugf("send config auth: %s", ctx.auth)
	log.Debugf("send config from: %s", ctx.from)
	log.Debugf("send config rcpts: %s", ctx.rcpts)
	log.Debugf("send config domain: %s", ctx.domain)

	warnSubject := composer.ShouldWarnSubject()
	warnAttachment := composer.ShouldWarnAttachment()
	if warnSubject || warnAttachment {
		var msg string
		switch {
		case warnSubject && warnAttachment:
			msg = "The subject is empty, and you may have forgotten an attachment."
		case warnSubject:
			msg = "The subject is empty."
		default:
			msg = "You may have forgotten an attachment."
		}

		prompt := app.NewPrompt(
			msg+" Abort send? [Y/n] ",
			func(text string) {
				if text == "n" || text == "N" {
					send(composer, ctx, header, tabName)
				}
			}, func(cmd string) ([]string, string) {
				if cmd == "" {
					return []string{"y", "n"}, ""
				}

				return nil, ""
			},
		)

		app.PushPrompt(prompt)
	} else {
		send(composer, ctx, header, tabName)
	}

	return nil
}

func send(composer *app.Composer, ctx sendCtx,
	header *mail.Header, tabName string,
) {
	// we don't want to block the UI thread while we are sending
	// so we do everything in a goroutine and hide the composer from the user
	app.RemoveTab(composer, false)
	app.PushStatus("Sending...", 10*time.Second)
	log.Debugf("send uri: %s", ctx.uri.String())

	// enter no-quit mode
	mode.NoQuit()

	var copyBuf bytes.Buffer // for the Sent folder content if CopyTo is set

	failCh := make(chan error)
	// writer
	go func() {
		defer log.PanicHandler()

		var sender io.WriteCloser
		var err error
		switch ctx.scheme {
		case "smtp":
			fallthrough
		case "smtp+insecure":
			fallthrough
		case "smtps":
			sender, err = newSmtpSender(ctx)
		case "jmap":
			sender, err = newJmapSender(composer, header, ctx)
		case "":
			sender, err = newSendmailSender(ctx)
		default:
			sender, err = nil, fmt.Errorf("unsupported scheme %v", ctx.scheme)
		}
		if err != nil {
			failCh <- errors.Wrap(err, "send:")
			return
		}

		var writer io.Writer = sender

		if ctx.copyto != "" && ctx.scheme != "jmap" {
			writer = io.MultiWriter(writer, &copyBuf)
		}
		err = composer.WriteMessage(header, writer)
		if err != nil {
			failCh <- err
			return
		}
		failCh <- sender.Close()
	}()

	// cleanup + copy to sent
	go func() {
		defer log.PanicHandler()

		// leave no-quit mode
		defer mode.NoQuitDone()

		err := <-failCh
		if err != nil {
			app.PushError(strings.ReplaceAll(err.Error(), "\n", " "))
			app.NewTab(composer, tabName)
			return
		}
		if ctx.copyto != "" && ctx.scheme != "jmap" {
			app.PushStatus("Copying to "+ctx.copyto, 10*time.Second)
			errch := copyToSent(ctx.copyto, copyBuf.Len(), &copyBuf,
				composer)
			err = <-errch
			if err != nil {
				errmsg := fmt.Sprintf(
					"message sent, but copying to %v failed: %v",
					ctx.copyto, err.Error())
				app.PushError(errmsg)
				composer.SetSent(ctx.archive)
				composer.Close()
				return
			}
		}
		app.PushStatus("Message sent.", 10*time.Second)
		composer.SetSent(ctx.archive)
		composer.Close()
	}()
}

func listRecipients(h *mail.Header) ([]*mail.Address, error) {
	var rcpts []*mail.Address
	for _, key := range []string{"to", "cc", "bcc"} {
		list, err := h.AddressList(key)
		if err != nil {
			return nil, err
		}
		rcpts = append(rcpts, list...)
	}
	return rcpts, nil
}

type sendCtx struct {
	uri     *url.URL
	scheme  string
	auth    string
	from    *mail.Address
	rcpts   []*mail.Address
	domain  string
	copyto  string
	archive string
}

func newSendmailSender(ctx sendCtx) (io.WriteCloser, error) {
	args := opt.SplitArgs(ctx.uri.Path)
	if len(args) == 0 {
		return nil, fmt.Errorf("no command specified")
	}
	bin := args[0]
	rs := make([]string, len(ctx.rcpts))
	for i := range ctx.rcpts {
		rs[i] = ctx.rcpts[i].Address
	}
	args = append(args[1:], rs...)
	cmd := exec.Command(bin, args...)
	s := &sendmailSender{cmd: cmd}
	var err error
	s.stdin, err = s.cmd.StdinPipe()
	if err != nil {
		return nil, errors.Wrap(err, "cmd.StdinPipe")
	}
	err = s.cmd.Start()
	if err != nil {
		return nil, errors.Wrap(err, "cmd.Start")
	}
	return s, nil
}

type sendmailSender struct {
	cmd   *exec.Cmd
	stdin io.WriteCloser
}

func (s *sendmailSender) Write(p []byte) (int, error) {
	return s.stdin.Write(p)
}

func (s *sendmailSender) Close() error {
	se := s.stdin.Close()
	ce := s.cmd.Wait()
	if se != nil {
		return se
	}
	return ce
}

func parseScheme(uri *url.URL) (scheme string, auth string, err error) {
	scheme = ""
	auth = "plain"
	if uri.Scheme != "" {
		parts := strings.Split(uri.Scheme, "+")
		switch len(parts) {
		case 1:
			scheme = parts[0]
		case 2:
			if parts[1] == "insecure" {
				scheme = uri.Scheme
			} else {
				scheme = parts[0]
				auth = parts[1]
			}
		case 3:
			scheme = parts[0] + "+" + parts[1]
			auth = parts[2]
		default:
			return "", "", fmt.Errorf("Unknown transfer protocol %s", uri.Scheme)
		}
	}
	return scheme, auth, nil
}

func newSaslClient(auth string, uri *url.URL) (sasl.Client, error) {
	var saslClient sasl.Client
	switch auth {
	case "":
		fallthrough
	case "none":
		saslClient = nil
	case "login":
		password, _ := uri.User.Password()
		saslClient = sasl.NewLoginClient(uri.User.Username(), password)
	case "plain":
		password, _ := uri.User.Password()
		saslClient = sasl.NewPlainClient("", uri.User.Username(), password)
	case "oauthbearer":
		q := uri.Query()
		oauth2 := &oauth2.Config{}
		if q.Get("token_endpoint") != "" {
			oauth2.ClientID = q.Get("client_id")
			oauth2.ClientSecret = q.Get("client_secret")
			oauth2.Scopes = []string{q.Get("scope")}
			oauth2.Endpoint.TokenURL = q.Get("token_endpoint")
		}
		password, _ := uri.User.Password()
		bearer := lib.OAuthBearer{
			OAuth2:  oauth2,
			Enabled: true,
		}
		if bearer.OAuth2.Endpoint.TokenURL != "" {
			token, err := bearer.ExchangeRefreshToken(password)
			if err != nil {
				return nil, err
			}
			password = token.AccessToken
		}
		saslClient = sasl.NewOAuthBearerClient(&sasl.OAuthBearerOptions{
			Username: uri.User.Username(),
			Token:    password,
		})
	case "xoauth2":
		q := uri.Query()
		oauth2 := &oauth2.Config{}
		if q.Get("token_endpoint") != "" {
			oauth2.ClientID = q.Get("client_id")
			oauth2.ClientSecret = q.Get("client_secret")
			oauth2.Scopes = []string{q.Get("scope")}
			oauth2.Endpoint.TokenURL = q.Get("token_endpoint")
		}
		password, _ := uri.User.Password()
		bearer := lib.Xoauth2{
			OAuth2:  oauth2,
			Enabled: true,
		}
		if bearer.OAuth2.Endpoint.TokenURL != "" {
			token, err := bearer.ExchangeRefreshToken(password)
			if err != nil {
				return nil, err
			}
			password = token.AccessToken
		}
		saslClient = lib.NewXoauth2Client(uri.User.Username(), password)
	default:
		return nil, fmt.Errorf("Unsupported auth mechanism %s", auth)
	}
	return saslClient, nil
}

type smtpSender struct {
	ctx  sendCtx
	conn *smtp.Client
	w    io.WriteCloser
}

func (s *smtpSender) Write(p []byte) (int, error) {
	return s.w.Write(p)
}

func (s *smtpSender) Close() error {
	we := s.w.Close()
	ce := s.conn.Close()
	if we != nil {
		return we
	}
	return ce
}

func newSmtpSender(ctx sendCtx) (io.WriteCloser, error) {
	var (
		err  error
		conn *smtp.Client
	)
	switch ctx.scheme {
	case "smtp":
		conn, err = connectSmtp(true, ctx.uri.Host, ctx.domain)
	case "smtp+insecure":
		conn, err = connectSmtp(false, ctx.uri.Host, ctx.domain)
	case "smtps":
		conn, err = connectSmtps(ctx.uri.Host)
	default:
		return nil, fmt.Errorf("not an smtp protocol %s", ctx.scheme)
	}

	if err != nil {
		return nil, errors.Wrap(err, "Connection failed")
	}

	saslclient, err := newSaslClient(ctx.auth, ctx.uri)
	if err != nil {
		conn.Close()
		return nil, err
	}
	if saslclient != nil {
		if err := conn.Auth(saslclient); err != nil {
			conn.Close()
			return nil, errors.Wrap(err, "conn.Auth")
		}
	}
	s := &smtpSender{
		ctx:  ctx,
		conn: conn,
	}
	if err := s.conn.Mail(s.ctx.from.Address, nil); err != nil {
		conn.Close()
		return nil, errors.Wrap(err, "conn.Mail")
	}
	for _, rcpt := range s.ctx.rcpts {
		if err := s.conn.Rcpt(rcpt.Address); err != nil {
			conn.Close()
			return nil, errors.Wrap(err, "conn.Rcpt")
		}
	}
	s.w, err = s.conn.Data()
	if err != nil {
		conn.Close()
		return nil, errors.Wrap(err, "conn.Data")
	}
	return s.w, nil
}

func connectSmtp(starttls bool, host string, domain string) (*smtp.Client, error) {
	serverName := host
	if !strings.ContainsRune(host, ':') {
		host += ":587" // Default to submission port
	} else {
		serverName = host[:strings.IndexRune(host, ':')]
	}
	conn, err := smtp.Dial(host)
	if err != nil {
		return nil, errors.Wrap(err, "smtp.Dial")
	}
	if domain != "" {
		err := conn.Hello(domain)
		if err != nil {
			return nil, errors.Wrap(err, "Hello")
		}
	}
	if starttls {
		if sup, _ := conn.Extension("STARTTLS"); !sup {
			err := errors.New("STARTTLS requested, but not supported " +
				"by this SMTP server. Is someone tampering with your " +
				"connection?")
			conn.Close()
			return nil, err
		}
		if err = conn.StartTLS(&tls.Config{
			ServerName: serverName,
		}); err != nil {
			conn.Close()
			return nil, errors.Wrap(err, "StartTLS")
		}
	}

	return conn, nil
}

func connectSmtps(host string) (*smtp.Client, error) {
	serverName := host
	if !strings.ContainsRune(host, ':') {
		host += ":465" // Default to smtps port
	} else {
		serverName = host[:strings.IndexRune(host, ':')]
	}
	conn, err := smtp.DialTLS(host, &tls.Config{
		ServerName: serverName,
	})
	if err != nil {
		return nil, errors.Wrap(err, "smtp.DialTLS")
	}
	return conn, nil
}

func newJmapSender(
	composer *app.Composer, header *mail.Header, ctx sendCtx,
) (io.WriteCloser, error) {
	var writer io.WriteCloser
	done := make(chan error)

	composer.Worker().PostAction(
		&types.StartSendingMessage{Header: header},
		func(msg types.WorkerMessage) {
			switch msg := msg.(type) {
			case *types.Done:
				return
			case *types.Unsupported:
				done <- fmt.Errorf("unsupported by worker")
			case *types.Error:
				done <- msg.Error
			case *types.MessageWriter:
				writer = msg.Writer
			default:
				done <- fmt.Errorf("unexpected worker message: %#v", msg)
			}
			close(done)
		},
	)

	err := <-done

	return writer, err
}

func copyToSent(dest string, n int, msg io.Reader, composer *app.Composer) <-chan error {
	errCh := make(chan error, 1)
	acct := composer.Account()
	if acct == nil {
		errCh <- errors.New("No account selected")
		return errCh
	}
	store := acct.Store()
	if store == nil {
		errCh <- errors.New("No message store selected")
		return errCh
	}
	store.Append(
		dest,
		models.SeenFlag,
		time.Now(),
		msg,
		n,
		func(msg types.WorkerMessage) {
			switch msg := msg.(type) {
			case *types.Done:
				errCh <- nil
			case *types.Error:
				errCh <- msg.Error
			}
		},
	)
	return errCh
}
