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
	"git.sr.ht/~rjarry/aerc/worker/imap/extensions"
	"git.sr.ht/~rjarry/aerc/worker/middleware"
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
	thread     *sortthread.ThreadClient
	sort       *sortthread.SortClient
	liststatus *extensions.ListStatusClient
}

type imapConfig struct {
	name              string
	scheme            string
	insecure          bool
	addr              string
	user              *url.Userinfo
	headers           []string
	headersExclude    []string
	folders           []string
	oauthBearer       lib.OAuthBearer
	xoauth2           lib.Xoauth2
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
	useXGMEXT          bool
	expungePolicy      int
}

type IMAPWorker struct {
	config imapConfig

	client    *imapClient
	selected  *imap.MailboxStatus
	updates   chan client.Update
	worker    types.WorkerInteractor
	seqMap    SeqMap
	expunger  *ExpungeHandler
	delimiter string

	idler    *idler
	observer *observer
	cache    *leveldb.DB

	caps *models.Capabilities

	threadAlgorithm sortthread.ThreadAlgorithm
	liststatus      bool

	noCheckMailBefore time.Time

	executeIdle chan struct{}
}

func NewIMAPWorker(worker *types.Worker) (types.Backend, error) {
	return &IMAPWorker{
		updates:           make(chan client.Update, 50),
		worker:            worker,
		selected:          &imap.MailboxStatus{},
		idler:             nil, // will be set in configure()
		observer:          nil, // will be set in configure()
		caps:              &models.Capabilities{},
		noCheckMailBefore: time.Now(),
		executeIdle:       make(chan struct{}),
	}, nil
}

func (w *IMAPWorker) newClient(c *client.Client) {
	c.Updates = nil
	w.client = &imapClient{
		c,
		sortthread.NewThreadClient(c),
		sortthread.NewSortClient(c),
		extensions.NewListStatusClient(c),
	}
	if w.idler != nil {
		w.idler.SetClient(w.client)
		c.Updates = w.updates
	}
	if w.observer != nil {
		w.observer.SetClient(w.client)
	}
	sort, err := w.client.sort.SupportSort()
	if err == nil && sort {
		w.caps.Sort = true
		w.worker.Debugf("Server Capability found: Sort")
	}
	for _, alg := range []sortthread.ThreadAlgorithm{sortthread.References, sortthread.OrderedSubject} {
		ok, err := w.client.Support(fmt.Sprintf("THREAD=%s", string(alg)))
		if err == nil && ok {
			w.threadAlgorithm = alg
			w.caps.Thread = true
			w.worker.Debugf("Server Capability found: Thread (algorithm: %s)", string(alg))
			break
		}
	}
	lStatus, err := w.client.liststatus.SupportListStatus()
	if err == nil && lStatus {
		w.liststatus = true
		w.caps.Extensions = append(w.caps.Extensions, "LIST-STATUS")
		w.worker.Debugf("Server Capability found: LIST-STATUS")
	}
	xgmext, err := w.client.Support("X-GM-EXT-1")
	if err == nil && xgmext && w.config.useXGMEXT {
		w.caps.Extensions = append(w.caps.Extensions, "X-GM-EXT-1")
		w.worker.Debugf("Server Capability found: X-GM-EXT-1")
		w.worker = middleware.NewGmailWorker(w.worker, w.client.Client)
	}
	if err == nil && !xgmext && w.config.useXGMEXT {
		w.worker.Infof("X-GM-EXT-1 requested, but it is not supported")
	}
}

func (w *IMAPWorker) handleMessage(msg types.WorkerMessage) error {
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
		// Reset the observer.
		w.observer.SetAutoReconnect(false)
		w.observer.Stop()
		w.observer.SetClient(nil)

		// Reset the idler, if any.
		if w.idler != nil {
			w.idler.SetClient(nil)
		}

		// Logout and reset the client.
		if w.client == nil || (w.client != nil && w.client.State() != imap.SelectedState) {
			reterr = errNotConnected
			w.client = nil
			break
		}
		if err := w.client.Logout(); err != nil {
			w.terminate()
			reterr = err
			break
		}
		w.client = nil

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
	case *types.MoveMessages:
		w.handleMoveMessages(msg)
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
	w.worker.Tracef("(= %T", update)
	switch update := update.(type) {
	case *client.MailboxUpdate:
		now := time.Now()
		// Since go-imap v1.2.1 gives *two* MailboxUpdate (one due to the
		// Unseen count and one due to the Recent count - see lines 413 and 431
		// in https://github.com/emersion/go-imap/blob/v1.2.1/client/client.go)
		// and each triggers a LIST-STATUS to the IMAP server, do a crude
		// deduping mechanism: ignore any MailboxUpdate received less than 20ms
		// after the last one on the very same worker.
		if now.After(w.noCheckMailBefore) {
			w.worker.PostAction(&types.CheckMail{
				Directories: []string{update.Mailbox.Name},
			}, nil)
		} else {
			w.worker.Debugf("Ignored duplicate MailboxUpdate")
		}
		w.noCheckMailBefore = now.Add(20 * time.Millisecond)
	case *client.MessageUpdate:
		msg := update.Message
		if msg.Uid == 0 {
			if uid, found := w.seqMap.Get(msg.SeqNum); !found {
				w.worker.Errorf("MessageUpdate unknown seqnum: %d", msg.SeqNum)
				return
			} else {
				msg.Uid = uid
			}
		}
		if w.expunger != nil && w.expunger.IsExpungingForDelete(msg.Uid) {
			// If we're deleting messages (vs. moving them), after we marked
			// them as Deleted and before expunging them (i.e. the worker's
			// ExpungeHandler is not nil), some IMAP servers will send a
			// MessageUpdate confirming that the messages have been marked as
			// Deleted. We should simply ignore those to avoid corrupting the
			// sequence.
			return
		}
		if int(msg.SeqNum) > w.seqMap.Size() {
			w.seqMap.Put(msg.Uid)
		}
		w.worker.PostMessage(&types.MessageInfo{
			Info: &models.MessageInfo{
				BodyStructure: translateBodyStructure(msg.BodyStructure),
				Envelope:      translateEnvelope(msg.Envelope),
				Flags:         translateImapFlags(msg.Flags),
				InternalDate:  msg.InternalDate,
				Uid:           models.Uint32ToUid(msg.Uid),
			},
			Unsolicited: true,
		}, nil)
	case *client.ExpungeUpdate:
		// We're notified of a message deletion. There are two cases:
		//  1. It's linked to a deletion from aerc, hence we have an expunger
		//     and find the sequence number there => we use the expunger to
		//     resolve the UID of the deleted message.
		//  2. Either we don't have an expunger or it does not contain the
		//     sequence number => we fallback to the actual sequence to resolve
		//     the UID of the deleted message.
		use_sequence := false
		var uid uint32 = 0
		found := false
		if w.expunger == nil {
			use_sequence = true
		} else if uid, found = w.expunger.PopSequenceNumber(update.SeqNum); !found {
			use_sequence = true
		}
		if use_sequence {
			uid, found = w.seqMap.Pop(update.SeqNum)
			if !found {
				uid = 0
				w.worker.Errorf("ExpungeUpdate unknown seqnum: %d", update.SeqNum)
			}
		}
		if uid != 0 {
			w.worker.PostMessage(&types.MessagesDeleted{
				Uids: []models.UID{models.Uint32ToUid(uid)},
			}, nil)
		}
	}
}

func (w *IMAPWorker) terminate() {
	if w.observer != nil {
		w.observer.Stop()
		w.observer.SetClient(nil)
	}

	if w.client != nil {
		w.client.Updates = nil
		if err := w.client.Terminate(); err != nil {
			w.worker.Errorf("could not terminate connection: %v", err)
		}
	}

	w.client = nil
	w.selected = &imap.MailboxStatus{}

	if w.idler != nil {
		w.idler.SetClient(nil)
	}
}

func (w *IMAPWorker) stopIdler() error {
	if w.idler == nil {
		return nil
	}

	if err := w.idler.Stop(); err != nil {
		w.terminate()
		w.observer.EmitIfNotConnected()
		w.worker.Errorf("idler stopped with error:%v", err)
		return err
	}

	return nil
}

func (w *IMAPWorker) startIdler() {
	if w.idler == nil {
		return
	}

	w.idler.Start()
}

func (w *IMAPWorker) Run() {
	for {
		select {
		case msg := <-w.worker.Actions():

			if err := w.stopIdler(); err != nil {
				w.worker.PostMessage(&types.Error{
					Message: types.RespondTo(msg),
					Error:   err,
				}, nil)
				break
			}
			w.worker.Tracef("ready to handle %T", msg)

			msg = w.worker.ProcessAction(msg)

			if err := w.handleMessage(msg); errors.Is(err, errUnsupported) {
				w.worker.PostMessage(&types.Unsupported{
					Message: types.RespondTo(msg),
				}, nil)
			} else if err != nil {
				w.worker.PostMessage(&types.Error{
					Message: types.RespondTo(msg),
					Error:   err,
				}, nil)
			}

			w.startIdler()

		case update := <-w.updates:
			w.handleImapUpdate(update)

		case <-w.executeIdle:
			w.idler.Execute()
		}
	}
}

func (w *IMAPWorker) Capabilities() *models.Capabilities {
	return w.caps
}

func (w *IMAPWorker) PathSeparator() string {
	if w.delimiter == "" {
		return "/"
	}
	return w.delimiter
}

func (w *IMAPWorker) BuildExpungeHandler(uids []uint32, forDelete bool) {
	w.expunger = NewExpungeHandler(w, uids, forDelete)
}
