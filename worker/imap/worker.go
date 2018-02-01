package imap

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
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
	messages chan types.WorkerMessage
	actions  chan types.WorkerMessage

	config struct {
		scheme   string
		insecure bool
		addr     string
		user     *url.Userinfo
	}

	client  *imapClient
	updates chan client.Update
	logger  *log.Logger
}

func NewIMAPWorker(logger *log.Logger) *IMAPWorker {
	return &IMAPWorker{
		messages: make(chan types.WorkerMessage, 50),
		actions:  make(chan types.WorkerMessage, 50),
		updates:  make(chan client.Update, 50),
		logger:   logger,
	}
}

func (w *IMAPWorker) GetMessages() chan types.WorkerMessage {
	return w.messages
}

func (w *IMAPWorker) PostAction(msg types.WorkerMessage) {
	w.actions <- msg
}

func (w *IMAPWorker) postMessage(msg types.WorkerMessage) {
	w.logger.Printf("=> %T\n", msg)
	w.messages <- msg
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
		w.postMessage(request)

		response := <-w.actions
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

		// TODO: don't idle right away
		go w.client.IdleWithFallback(nil, 0)
	default:
		return errUnsupported
	}
	return nil
}

func (w *IMAPWorker) Run() {
	for {
		select {
		case msg := <-w.actions:
			w.logger.Printf("<= %T\n", msg)
			if err := w.handleMessage(msg); err == errUnsupported {
				w.postMessage(types.Unsupported{
					Message: types.RespondTo(msg),
				})
			} else if err != nil {
				w.postMessage(types.Error{
					Message: types.RespondTo(msg),
					Error:   err,
				})
			} else {
				w.postMessage(types.Ack{
					Message: types.RespondTo(msg),
				})
			}
		case update := <-w.updates:
			w.logger.Printf("[= %T", update)
		}
	}
}
