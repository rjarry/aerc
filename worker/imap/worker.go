package imap

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/url"
	"strings"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap-idle"
	"github.com/emersion/go-imap/client"

	"git.sr.ht/~sircmpwn/aerc2/worker/types"
)

var errUnsupported = fmt.Errorf("unsupported command")

type imapClient struct {
	*client.Client
	*idle.IdleClient
}

type IMAPWorker struct {
	config struct {
		scheme   string
		insecure bool
		addr     string
		user     *url.Userinfo
	}

	worker  *types.Worker
	client  *imapClient
	updates chan client.Update
}

func NewIMAPWorker(worker *types.Worker) *IMAPWorker {
	return &IMAPWorker{
		worker:  worker,
		updates: make(chan client.Update, 50),
	}
}

func (w *IMAPWorker) verifyPeerCert(msg types.WorkerMessage) func(
	rawCerts [][]byte, _ [][]*x509.Certificate) error {

	return func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
		pool := x509.NewCertPool()
		for _, rawCert := range rawCerts {
			cert, err := x509.ParseCertificate(rawCert)
			if err != nil {
				return err
			}
			pool.AddCert(cert)
		}

		request := types.ApproveCertificate{
			Message:  types.RespondTo(msg),
			CertPool: pool,
		}
		w.worker.PostMessage(request, nil)

		response := <-w.worker.Actions
		if response.InResponseTo() != request {
			return fmt.Errorf("Expected UI to answer cert request")
		}
		switch response.(type) {
		case types.Ack:
			return nil
		case types.Disconnect:
			return fmt.Errorf("UI rejected certificate")
		default:
			return fmt.Errorf("Expected UI to answer cert request")
		}
	}
}

func (w *IMAPWorker) handleMessage(msg types.WorkerMessage) error {
	switch msg := msg.(type) {
	case types.Ping:
	case types.Unsupported:
		// No-op
	case types.Configure:
		u, err := url.Parse(msg.Config.Source)
		if err != nil {
			return err
		}

		w.config.scheme = u.Scheme
		if strings.HasSuffix(w.config.scheme, "+insecure") {
			w.config.scheme = strings.TrimSuffix(w.config.scheme, "+insecure")
			w.config.insecure = true
		}

		w.config.addr = u.Host
		if !strings.ContainsRune(w.config.addr, ':') {
			w.config.addr += ":" + u.Scheme
		}

		w.config.scheme = u.Scheme
		w.config.user = u.User
	case types.Connect:
		var (
			c   *client.Client
			err error
		)
		tlsConfig := &tls.Config{
			InsecureSkipVerify:    true,
			VerifyPeerCertificate: w.verifyPeerCert(&msg),
		}
		switch w.config.scheme {
		case "imap":
			c, err = client.Dial(w.config.addr)
			if err != nil {
				return err
			}

			if !w.config.insecure {
				if err := c.StartTLS(tlsConfig); err != nil {
					return err
				}
			}
		case "imaps":
			c, err = client.DialTLS(w.config.addr, tlsConfig)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("Unknown IMAP scheme %s", w.config.scheme)
		}

		if w.config.user != nil {
			username := w.config.user.Username()
			password, hasPassword := w.config.user.Password()
			if !hasPassword {
				// TODO: ask password
			}
			if err := c.Login(username, password); err != nil {
				return err
			}
		}

		if _, err := c.Select(imap.InboxName, false); err != nil {
			return err
		}

		c.Updates = w.updates
		w.client = &imapClient{c, idle.NewClient(c)}
	case types.ListDirectories:
		w.handleListDirectories(msg)
	default:
		return errUnsupported
	}
	return nil
}

func (w *IMAPWorker) Run() {
	for {
		select {
		case msg := <-w.worker.Actions:
			msg = w.worker.ProcessAction(msg)
			if err := w.handleMessage(msg); err == errUnsupported {
				w.worker.PostMessage(types.Unsupported{
					Message: types.RespondTo(msg),
				}, nil)
			} else if err != nil {
				w.worker.PostMessage(types.Error{
					Message: types.RespondTo(msg),
					Error:   err,
				}, nil)
			} else {
				w.worker.PostMessage(types.Ack{
					Message: types.RespondTo(msg),
				}, nil)
			}
		case update := <-w.updates:
			w.worker.Logger.Printf("(= %T", update)
		}
	}
}
