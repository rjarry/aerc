package imap

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	sortthread "github.com/emersion/go-imap-sortthread"
	"github.com/emersion/go-imap/client"
	"golang.org/x/oauth2"

	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/handlers"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

func init() {
	handlers.RegisterWorkerFactory("imap", NewIMAPWorker)
	handlers.RegisterWorkerFactory("imaps", NewIMAPWorker)
}

var errUnsupported = fmt.Errorf("unsupported command")

type imapClient struct {
	*client.Client
	thread *sortthread.ThreadClient
	sort   *sortthread.SortClient
}

type IMAPWorker struct {
	config struct {
		scheme      string
		insecure    bool
		addr        string
		user        *url.Userinfo
		folders     []string
		oauthBearer lib.OAuthBearer
		// tcp connection parameters
		connection_timeout time.Duration
		keepalive_period   time.Duration
		keepalive_probes   int
		keepalive_interval int
	}

	client   *imapClient
	idleStop chan struct{}
	idleDone chan error
	selected *imap.MailboxStatus
	updates  chan client.Update
	worker   *types.Worker
	// Map of sequence numbers to UIDs, index 0 is seq number 1
	seqMap        []uint32
	done          chan struct{}
	autoReconnect bool
}

func NewIMAPWorker(worker *types.Worker) (types.Backend, error) {
	return &IMAPWorker{
		idleDone: make(chan error),
		updates:  make(chan client.Update, 50),
		worker:   worker,
		selected: &imap.MailboxStatus{},
	}, nil
}

func (w *IMAPWorker) handleMessage(msg types.WorkerMessage) error {
	if w.client != nil && w.client.State() == imap.SelectedState {
		close(w.idleStop)
		if err := <-w.idleDone; err != nil {
			w.worker.PostMessage(&types.Error{Error: err}, nil)
		}
	}
	defer func() {
		if w.client != nil && w.client.State() == imap.SelectedState {
			w.idleStop = make(chan struct{})
			go func() {
				w.idleDone <- w.client.Idle(w.idleStop, &client.IdleOptions{0, 0})
			}()
		}
	}()

	checkConn := func() {
		w.stopConnectionObserver()
		w.startConnectionObserver()
	}

	var reterr error // will be returned at the end, needed to support idle

	switch msg := msg.(type) {
	case *types.Unsupported:
		// No-op
	case *types.Configure:
		u, err := url.Parse(msg.Config.Source)
		if err != nil {
			reterr = err
			break
		}

		w.config.scheme = u.Scheme
		if strings.HasSuffix(w.config.scheme, "+insecure") {
			w.config.scheme = strings.TrimSuffix(w.config.scheme, "+insecure")
			w.config.insecure = true
		}

		if strings.HasSuffix(w.config.scheme, "+oauthbearer") {
			w.config.scheme = strings.TrimSuffix(w.config.scheme, "+oauthbearer")
			w.config.oauthBearer.Enabled = true
			q := u.Query()

			oauth2 := &oauth2.Config{}
			if q.Get("token_endpoint") != "" {
				oauth2.ClientID = q.Get("client_id")
				oauth2.ClientSecret = q.Get("client_secret")
				oauth2.Scopes = []string{q.Get("scope")}
				oauth2.Endpoint.TokenURL = q.Get("token_endpoint")
			}
			w.config.oauthBearer.OAuth2 = oauth2
		}

		w.config.addr = u.Host
		if !strings.ContainsRune(w.config.addr, ':') {
			w.config.addr += ":" + w.config.scheme
		}

		w.config.user = u.User
		w.config.folders = msg.Config.Folders
		w.config.connection_timeout = 30 * time.Second
		w.config.keepalive_period = 0 * time.Second
		w.config.keepalive_probes = 3
		w.config.keepalive_interval = 3
		for key, value := range msg.Config.Params {
			switch key {
			case "connection-timeout":
				val, err := time.ParseDuration(value)
				if err != nil || val < 0 {
					reterr = fmt.Errorf(
						"invalid connection-timeout value %v: %v",
						value, err)
					break
				}
				w.config.connection_timeout = val
			case "keepalive-period":
				val, err := time.ParseDuration(value)
				if err != nil || val < 0 {
					reterr = fmt.Errorf(
						"invalid keepalive-period value %v: %v",
						value, err)
					break
				}
				w.config.keepalive_period = val
			case "keepalive-probes":
				val, err := strconv.Atoi(value)
				if err != nil || val < 0 {
					reterr = fmt.Errorf(
						"invalid keepalive-probes value %v: %v",
						value, err)
					break
				}
				w.config.keepalive_probes = val
			case "keepalive-interval":
				val, err := time.ParseDuration(value)
				if err != nil || val < 0 {
					reterr = fmt.Errorf(
						"invalid keepalive-interval value %v: %v",
						value, err)
					break
				}
				w.config.keepalive_interval = int(val.Seconds())
			}
		}
	case *types.Connect:
		if w.client != nil && w.client.State() == imap.SelectedState {
			if !w.autoReconnect {
				w.autoReconnect = true
				checkConn()
			}
			reterr = fmt.Errorf("Already connected")
			break
		}

		w.autoReconnect = true
		c, err := w.connect()
		if err != nil {
			reterr = err
			break
		}

		w.stopConnectionObserver()

		c.Updates = w.updates
		w.client = &imapClient{c, sortthread.NewThreadClient(c), sortthread.NewSortClient(c)}

		w.startConnectionObserver()

		w.worker.PostMessage(&types.Done{types.RespondTo(msg)}, nil)
	case *types.Reconnect:
		if !w.autoReconnect {
			reterr = fmt.Errorf("auto-reconnect is disabled; run connect to enable it")
			break
		}
		c, err := w.connect()
		if err != nil {
			checkConn()
			reterr = err
			break
		}

		w.stopConnectionObserver()

		c.Updates = w.updates
		w.client = &imapClient{c, sortthread.NewThreadClient(c), sortthread.NewSortClient(c)}

		w.startConnectionObserver()

		w.worker.PostMessage(&types.Done{types.RespondTo(msg)}, nil)
	case *types.Disconnect:
		w.autoReconnect = false
		w.stopConnectionObserver()
		if w.client == nil || w.client.State() != imap.SelectedState {
			reterr = fmt.Errorf("Not connected")
			break
		}

		if err := w.client.Logout(); err != nil {
			reterr = err
			break
		}
		w.worker.PostMessage(&types.Done{types.RespondTo(msg)}, nil)
	case *types.ListDirectories:
		w.handleListDirectories(msg)
	case *types.OpenDirectory:
		w.handleOpenDirectory(msg)
	case *types.FetchDirectoryContents:
		w.handleFetchDirectoryContents(msg)
	case *types.FetchDirectoryThreaded:
		w.handleDirectoryThreaded(msg)
	case *types.CreateDirectory:
		w.handleCreateDirectory(msg)
	case *types.RemoveDirectory:
		w.handleRemoveDirectory(msg)
	case *types.FetchMessageHeaders:
		w.handleFetchMessageHeaders(msg)
	case *types.FetchMessageBodyPart:
		w.handleFetchMessageBodyPart(msg)
	case *types.FetchFullMessages:
		w.handleFetchFullMessages(msg)
	case *types.DeleteMessages:
		w.handleDeleteMessages(msg)
	case *types.FlagMessages:
		w.handleFlagMessages(msg)
	case *types.AnsweredMessages:
		w.handleAnsweredMessages(msg)
	case *types.CopyMessages:
		w.handleCopyMessages(msg)
	case *types.AppendMessage:
		w.handleAppendMessage(msg)
	case *types.SearchDirectory:
		w.handleSearchDirectory(msg)
	default:
		reterr = errUnsupported
	}

	return reterr
}

func (w *IMAPWorker) handleImapUpdate(update client.Update) {
	w.worker.Logger.Printf("(= %T", update)
	switch update := update.(type) {
	case *client.MailboxUpdate:
		status := update.Mailbox
		if w.selected.Name == status.Name {
			w.selected = status
		}
		w.worker.PostMessage(&types.DirectoryInfo{
			Info: &models.DirectoryInfo{
				Flags:    status.Flags,
				Name:     status.Name,
				ReadOnly: status.ReadOnly,

				Exists: int(status.Messages),
				Recent: int(status.Recent),
				Unseen: int(status.Unseen),
			},
		}, nil)
	case *client.MessageUpdate:
		msg := update.Message
		if msg.Uid == 0 {
			msg.Uid = w.seqMap[msg.SeqNum-1]
		}
		w.worker.PostMessage(&types.MessageInfo{
			Info: &models.MessageInfo{
				BodyStructure: translateBodyStructure(msg.BodyStructure),
				Envelope:      translateEnvelope(msg.Envelope),
				Flags:         translateImapFlags(msg.Flags),
				InternalDate:  msg.InternalDate,
				Uid:           msg.Uid,
			},
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

func (w *IMAPWorker) startConnectionObserver() {
	go func() {
		select {
		case <-w.client.LoggedOut():
			if w.autoReconnect {
				w.worker.PostMessage(&types.ConnError{
					Error: fmt.Errorf("imap: logged out"),
				}, nil)
			}
		case <-w.done:
			return
		}
	}()
}

func (w *IMAPWorker) stopConnectionObserver() {
	if w.done != nil {
		close(w.done)
	}
	w.done = make(chan struct{})
}

func (w *IMAPWorker) connect() (*client.Client, error) {
	var (
		conn *net.TCPConn
		c    *client.Client
	)

	addr, err := net.ResolveTCPAddr("tcp", w.config.addr)
	if err != nil {
		return nil, err
	}

	conn, err = net.DialTCP("tcp", nil, addr)
	if err != nil {
		return nil, err
	}

	if w.config.connection_timeout > 0 {
		end := time.Now().Add(w.config.connection_timeout)
		err = conn.SetDeadline(end)
		if err != nil {
			return nil, err
		}
	}
	if w.config.keepalive_period > 0 {
		err = w.setKeepaliveParameters(conn)
		if err != nil {
			return nil, err
		}
	}

	serverName, _, _ := net.SplitHostPort(w.config.addr)
	tlsConfig := &tls.Config{ServerName: serverName}

	switch w.config.scheme {
	case "imap":
		c, err = client.New(conn)
		if err != nil {
			return nil, err
		}
		if !w.config.insecure {
			if err = c.StartTLS(tlsConfig); err != nil {
				return nil, err
			}
		}
	case "imaps":
		tlsConn := tls.Client(conn, tlsConfig)
		c, err = client.New(tlsConn)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("Unknown IMAP scheme %s", w.config.scheme)
	}

	c.ErrorLog = w.worker.Logger

	if w.config.user != nil {
		username := w.config.user.Username()
		password, hasPassword := w.config.user.Password()
		if !hasPassword {
			// TODO: ask password
		}

		if w.config.oauthBearer.Enabled {
			if err := w.config.oauthBearer.Authenticate(
				username, password, c); err != nil {
				return nil, err
			}
		} else if err := c.Login(username, password); err != nil {
			return nil, err
		}
	}

	c.SetDebug(w.worker.Logger.Writer())

	if _, err := c.Select(imap.InboxName, false); err != nil {
		return nil, err
	}

	return c, nil
}

// Set additional keepalive parameters.
// Uses new interfaces introduced in Go1.11, which let us get connection's file
// descriptor, without blocking, and therefore without uncontrolled spawning of
// threads (not goroutines, actual threads).
func (w *IMAPWorker) setKeepaliveParameters(conn *net.TCPConn) error {
	err := conn.SetKeepAlive(true)
	if err != nil {
		return err
	}
	// Idle time before sending a keepalive probe
	err = conn.SetKeepAlivePeriod(w.config.keepalive_period)
	if err != nil {
		return err
	}
	rawConn, e := conn.SyscallConn()
	if e != nil {
		return e
	}
	err = rawConn.Control(func(fdPtr uintptr) {
		fd := int(fdPtr)
		// Max number of probes before failure
		err := lib.SetTcpKeepaliveProbes(fd, w.config.keepalive_probes)
		if err != nil {
			w.worker.Logger.Printf(
				"cannot set tcp keepalive probes: %v\n", err)
		}
		// Wait time after an unsuccessful probe
		err = lib.SetTcpKeepaliveInterval(fd, w.config.keepalive_interval)
		if err != nil {
			w.worker.Logger.Printf(
				"cannot set tcp keepalive interval: %v\n", err)
		}
	})
	return err
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
