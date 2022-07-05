package imap

import (
	"fmt"
	"net/url"
	"time"

	"github.com/emersion/go-imap"
	sortthread "github.com/emersion/go-imap-sortthread"
	"github.com/emersion/go-imap/client"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"

	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/handlers"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

func init() {
	handlers.RegisterWorkerFactory("imap", NewIMAPWorker)
	handlers.RegisterWorkerFactory("imaps", NewIMAPWorker)
}

var (
	errUnsupported      = fmt.Errorf("unsupported command")
	errClientNotReady   = fmt.Errorf("client not ready")
	errNotConnected     = fmt.Errorf("not connected")
	errAlreadyConnected = fmt.Errorf("already connected")
)

type imapClient struct {
	*client.Client
	thread *sortthread.ThreadClient
	sort   *sortthread.SortClient
}

type imapConfig struct {
	scheme            string
	insecure          bool
	addr              string
	user              *url.Userinfo
	folders           []string
	oauthBearer       lib.OAuthBearer
	idle_timeout      time.Duration
	idle_debounce     time.Duration
	reconnect_maxwait time.Duration
	// tcp connection parameters
	connection_timeout time.Duration
	keepalive_period   time.Duration
	keepalive_probes   int
	keepalive_interval int
	cacheEnabled       bool
	cacheMaxAge        time.Duration
}

type IMAPWorker struct {
	config imapConfig

	client   *imapClient
	selected *imap.MailboxStatus
	updates  chan client.Update
	worker   *types.Worker
	seqMap   SeqMap

	idler    *idler
	observer *observer
	cache    *leveldb.DB

	caps *models.Capabilities
}

func NewIMAPWorker(worker *types.Worker) (types.Backend, error) {
	return &IMAPWorker{
		updates:  make(chan client.Update, 50),
		worker:   worker,
		selected: &imap.MailboxStatus{},
		idler:    newIdler(imapConfig{}, worker),
		observer: newObserver(imapConfig{}, worker),
		caps:     &models.Capabilities{},
	}, nil
}

func (w *IMAPWorker) newClient(c *client.Client) {
	c.Updates = w.updates
	w.client = &imapClient{c, sortthread.NewThreadClient(c), sortthread.NewSortClient(c)}
	w.idler.SetClient(w.client)
	w.observer.SetClient(w.client)
	sort, err := w.client.sort.SupportSort()
	if err == nil && sort {
		w.caps.Sort = true
		w.worker.Logger.Println("Server Capability found: Sort")
	}
	thread, err := w.client.thread.SupportThread()
	if err == nil && thread {
		w.caps.Thread = true
		w.worker.Logger.Println("Server Capability found: Thread")
	}
}

func (w *IMAPWorker) handleMessage(msg types.WorkerMessage) error {
	defer func() {
		w.idler.Start()
	}()
	if err := w.idler.Stop(); err != nil {
		return err
	}

	var reterr error // will be returned at the end, needed to support idle

	// when client is nil allow only certain messages to be handled
	if w.client == nil {
		switch msg.(type) {
		case *types.Connect, *types.Reconnect, *types.Disconnect, *types.Configure:
		default:
			return errClientNotReady
		}
	}

	// set connection timeout for calls to imap server
	if w.client != nil {
		w.client.Timeout = w.config.connection_timeout
	}

	switch msg := msg.(type) {
	case *types.Unsupported:
		// No-op
	case *types.Configure:
		reterr = w.handleConfigure(msg)
	case *types.Connect:
		if w.client != nil && w.client.State() == imap.SelectedState {
			if !w.observer.AutoReconnect() {
				w.observer.SetAutoReconnect(true)
				w.observer.EmitIfNotConnected()
			}
			reterr = errAlreadyConnected
			break
		}

		w.observer.SetAutoReconnect(true)
		c, err := w.connect()
		if err != nil {
			w.observer.EmitIfNotConnected()
			reterr = err
			break
		}

		w.newClient(c)

		w.worker.PostMessage(&types.Done{Message: types.RespondTo(msg)}, nil)
	case *types.Reconnect:
		if !w.observer.AutoReconnect() {
			reterr = fmt.Errorf("auto-reconnect is disabled; run connect to enable it")
			break
		}
		c, err := w.connect()
		if err != nil {
			errReconnect := w.observer.DelayedReconnect()
			reterr = errors.Wrap(errReconnect, err.Error())
			break
		}

		w.newClient(c)

		w.worker.PostMessage(&types.Done{Message: types.RespondTo(msg)}, nil)
	case *types.Disconnect:
		w.observer.SetAutoReconnect(false)
		w.observer.Stop()
		if w.client == nil || w.client.State() != imap.SelectedState {
			reterr = errNotConnected
			break
		}

		if err := w.client.Logout(); err != nil {
			reterr = err
			break
		}
		w.worker.PostMessage(&types.Done{Message: types.RespondTo(msg)}, nil)
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
	case *types.FetchMessageFlags:
		w.handleFetchMessageFlags(msg)
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
	case *types.CheckMail:
		w.handleCheckMailMessage(msg)
	default:
		reterr = errUnsupported
	}

	// we don't want idle to timeout, so set timeout to zero
	if w.client != nil {
		w.client.Timeout = 0
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
				Caps:   w.caps,
			},
		}, nil)
	case *client.MessageUpdate:
		msg := update.Message
		if msg.Uid == 0 {
			if uid, found := w.seqMap.Get(msg.SeqNum); !found {
				w.worker.Logger.Printf("MessageUpdate unknown seqnum: %v", msg.SeqNum)
				return
			} else {
				msg.Uid = uid
			}
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
		if uid, found := w.seqMap.Pop(update.SeqNum); !found {
			w.worker.Logger.Printf("ExpungeUpdate unknown seqnum: %v", update.SeqNum)
		} else {
			w.worker.PostMessage(&types.MessagesDeleted{
				Uids: []uint32{uid},
			}, nil)
		}
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
