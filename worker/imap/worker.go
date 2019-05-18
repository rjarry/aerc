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

	"git.sr.ht/~sircmpwn/aerc/worker/types"
)

var errUnsupported = fmt.Errorf("unsupported command")

type imapClient struct {
	*client.Client
	idle *idle.IdleClient
}

type IMAPWorker struct {
	config struct {
		scheme   string
		insecure bool
		addr     string
		user     *url.Userinfo
	}

	client   *imapClient
	idleStop chan struct{}
	idleDone chan error
	selected imap.MailboxStatus
	updates  chan client.Update
	worker   *types.Worker
	// Map of sequence numbers to UIDs, index 0 is seq number 1
	seqMap []uint32
}

func NewIMAPWorker(worker *types.Worker) *IMAPWorker {
	return &IMAPWorker{
		idleDone: make(chan error),
		updates:  make(chan client.Update, 50),
		worker:   worker,
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

		request := &types.CertificateApprovalRequest{
			Message:  types.RespondTo(msg),
			CertPool: pool,
		}
		w.worker.PostMessage(request, nil)

		response := <-w.worker.Actions
		if response.InResponseTo() != request {
			return fmt.Errorf("Expected UI to respond to cert request")
		}
		if approval, ok := response.(*types.ApproveCertificate); !ok {
			return fmt.Errorf("Expected UI to send certificate approval")
		} else {
			if approval.Approved {
				return nil
			} else {
				return fmt.Errorf("UI rejected certificate")
			}
		}
	}
}

func (w *IMAPWorker) handleMessage(msg types.WorkerMessage) error {
	if w.idleStop != nil {
		close(w.idleStop)
		if err := <-w.idleDone; err != nil {
			w.worker.PostMessage(&types.Error{Error: err}, nil)
		}
	}

	switch msg := msg.(type) {
	case *types.Unsupported:
		// No-op
	case *types.Configure:
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
	case *types.Connect:
		var (
			c   *client.Client
			err error
		)
		tlsConfig := &tls.Config{
			InsecureSkipVerify:    true,
			VerifyPeerCertificate: w.verifyPeerCert(msg),
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

		c.SetDebug(w.worker.Logger.Writer())

		if _, err := c.Select(imap.InboxName, false); err != nil {
			return err
		}

		c.Updates = w.updates
		w.client = &imapClient{c, idle.NewClient(c)}
		w.worker.PostMessage(&types.Done{types.RespondTo(msg)}, nil)
	case *types.ListDirectories:
		w.handleListDirectories(msg)
	case *types.OpenDirectory:
		w.handleOpenDirectory(msg)
	case *types.FetchDirectoryContents:
		w.handleFetchDirectoryContents(msg)
	case *types.FetchMessageHeaders:
		w.handleFetchMessageHeaders(msg)
	case *types.FetchMessageBodyPart:
		w.handleFetchMessageBodyPart(msg)
	case *types.FetchFullMessages:
		w.handleFetchFullMessages(msg)
	case *types.DeleteMessages:
		w.handleDeleteMessages(msg)
	case *types.CopyMessages:
		w.handleCopyMessages(msg)
	case *types.AppendMessage:
		w.handleAppendMessage(msg)
	default:
		return errUnsupported
	}

	if w.idleStop != nil {
		w.idleStop = make(chan struct{})
		go func() {
			w.idleDone <- w.client.idle.IdleWithFallback(w.idleStop, 0)
		}()
	}
	return nil
}

func (w *IMAPWorker) handleImapUpdate(update client.Update) {
	w.worker.Logger.Printf("(= %T", update)
	switch update := update.(type) {
	case *client.MailboxUpdate:
		status := update.Mailbox
		if w.selected.Name == status.Name {
			w.selected = *status
		}
		w.worker.PostMessage(&types.DirectoryInfo{
			Flags:    status.Flags,
			Name:     status.Name,
			ReadOnly: status.ReadOnly,

			Exists: int(status.Messages),
			Recent: int(status.Recent),
			Unseen: int(status.Unseen),
		}, nil)
	case *client.ExpungeUpdate:
		i := update.SeqNum - 1
		uid := w.seqMap[i]
		w.seqMap = append(w.seqMap[:i], w.seqMap[i+1:]...)
		w.worker.PostMessage(&types.MessagesDeleted{
			Uids: []uint32{uid},
		}, nil)
	}
}

func (w *IMAPWorker) Run() {
	for {
		select {
		case msg := <-w.worker.Actions:
			msg = w.worker.ProcessAction(msg)
			if err := w.handleMessage(msg); err == errUnsupported {
				w.worker.PostMessage(&types.Unsupported{
					Message: types.RespondTo(msg),
				}, nil)
			} else if err != nil {
				w.worker.PostMessage(&types.Error{
					Message: types.RespondTo(msg),
					Error:   err,
				}, nil)
			}
		case update := <-w.updates:
			w.handleImapUpdate(update)
		}
	}
}
