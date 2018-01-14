package imap

import (
	"fmt"
	"net/url"
	"strings"

	"git.sr.ht/~sircmpwn/aerc2/worker/types"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-imap-idle"
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
}

func NewIMAPWorker() *IMAPWorker {
	return &IMAPWorker{
		messages: make(chan types.WorkerMessage, 50),
		actions:  make(chan types.WorkerMessage, 50),
		updates:  make(chan client.Update, 50),
	}
}

func (w *IMAPWorker) GetMessages() chan types.WorkerMessage {
	return w.messages
}

func (w *IMAPWorker) PostAction(msg types.WorkerMessage) {
	w.actions <- msg
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
		// TODO: populate TLS config

		var (
			c   *client.Client
			err error
		)
		switch w.config.scheme {
		case "imap":
			c, err = client.Dial(w.config.addr)
			if err != nil {
				return err
			}

			if !w.config.insecure {
				if err := c.StartTLS(nil); err != nil {
					return err
				}
			}
		case "imaps":
			c, err = client.DialTLS(w.config.addr, nil)
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
			fmt.Printf("<= %T\n", msg)
			if err := w.handleMessage(msg); err == errUnsupported {
				w.messages <- types.Unsupported{
					Message: types.RespondTo(msg),
				}
			} else if err != nil {
				w.messages <- types.Error{
					Message: types.RespondTo(msg),
					Error:   err,
				}
			} else {
				w.messages <- types.Ack{
					Message: types.RespondTo(msg),
				}
			}
		case update := <-w.updates:
			fmt.Printf("<= %T\n", update)
		}
	}
}
