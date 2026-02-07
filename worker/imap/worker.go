package imap

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/emersion/go-imap"
	sortthread "github.com/emersion/go-imap-sortthread"
	"github.com/emersion/go-imap/client"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"

	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/handlers"
	"git.sr.ht/~rjarry/aerc/worker/imap/extensions"
	"git.sr.ht/~rjarry/aerc/worker/imap/extensions/xgmext"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

func init() {
	handlers.RegisterWorkerFactory("imap", NewIMAPWorker)
	handlers.RegisterWorkerFactory("imaps", NewIMAPWorker)
}

var (
	errClientNotReady   = fmt.Errorf("client not ready")
	errNotConnected     = fmt.Errorf("not connected")
	errAlreadyConnected = fmt.Errorf("already connected")
)

type imapProvider uint32

const (
	Unknown imapProvider = iota
	GMail
	Proton
	Office365
	Zoho
	FastMail
	iCloud
)

type imapClient struct {
	*client.Client
	thread     *sortthread.ThreadClient
	sort       *sortthread.SortClient
	liststatus *extensions.ListStatusClient
	xgmext     *xgmext.XGMExtClient
}

type imapConfig struct {
	name           string
	url            *url.URL
	provider       imapProvider
	headers        []string
	headersExclude []string
	folders        []string
	// tcp connection parameters
	connection_timeout time.Duration
	keepalive_period   time.Duration
	keepalive_probes   int
	keepalive_interval int
	cacheEnabled       bool
	cacheMaxAge        time.Duration
	expungePolicy      int
	checkMail          time.Duration
	debugLogPath       string
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
	w := &IMAPWorker{
		updates:           make(chan client.Update, 50),
		worker:            worker,
		selected:          &imap.MailboxStatus{},
		observer:          newObserver(worker),
		caps:              &models.Capabilities{},
		noCheckMailBefore: time.Now(),
		executeIdle:       make(chan struct{}),
	}
	w.idler = newIdler(worker, w.executeIdle)
	return w, nil
}

func (w *IMAPWorker) newClient(c *client.Client) {
	c.Updates = nil
	w.client = &imapClient{
		Client:     c,
		thread:     sortthread.NewThreadClient(c),
		sort:       sortthread.NewSortClient(c),
		liststatus: extensions.NewListStatusClient(c),
		xgmext:     xgmext.NewXGMExtClient(c),
	}
	w.idler.SetClient(w.client)
	c.Updates = w.updates
	w.observer.SetClient(w.client)
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
	if err == nil && xgmext {
		w.caps.Extensions = append(w.caps.Extensions, "X-GM-EXT-1")
		w.worker.Debugf("Server Capability found: X-GM-EXT-1")
		if w.config.provider != GMail {
			w.worker.Warnf("Provider detection issue; setting to GMail since X-GM-EXT-1 is supported")
			w.config.provider = GMail
		}
	}
}

func (w *IMAPWorker) handleMessage(msg types.WorkerMessage) error {
	// when client is nil allow only certain messages to be handled
	if w.client == nil {
		switch msg.(type) {
		case *types.Connect, *types.Reconnect, *types.Disconnect, *types.Configure:
			break
		default:
			return errClientNotReady
		}
	}

	switch msg := msg.(type) {
	case *types.Unsupported:
		return types.ErrNoop
	case *types.Configure:
		return w.handleConfigure(msg)
	case *types.Connect:
		if w.client != nil && w.client.State() == imap.SelectedState {
			return errAlreadyConnected
		}

		c, err := w.connect()
		if err != nil {
			w.observer.EmitIfNotConnected()
			return err
		}

		w.newClient(c)

		return nil
	case *types.Reconnect:
		c, err := w.connect()
		if err != nil {
			// Send ConnError to trigger retry from account.go
			// (consolidates reconnection logic with other backends)
			w.worker.PostMessage(&types.ConnError{Error: err}, nil)
			break
		}

		w.newClient(c)

		return nil
	case *types.Disconnect:
		// Reset the observer.
		w.observer.Stop()
		w.observer.SetClient(nil)

		// Reset the idler, if any.
		w.idler.SetClient(nil)

		// Logout and reset the client.
		if w.client == nil || (w.client != nil && w.client.State() != imap.SelectedState) {
			w.client = nil
			return errNotConnected
		}
		if err := w.client.Logout(); err != nil {
			w.terminate()
			return err
		}
		w.client = nil

		return nil
	case *types.ListDirectories:
		return w.handleListDirectories(msg)
	case *types.OpenDirectory:
		return w.handleOpenDirectory(msg)
	case *types.FetchDirectoryContents:
		return w.handleFetchDirectoryContents(msg)
	case *types.FetchDirectoryThreaded:
		return w.handleDirectoryThreaded(msg)
	case *types.CreateDirectory:
		return w.handleCreateDirectory(msg)
	case *types.RemoveDirectory:
		return w.handleRemoveDirectory(msg)
	case *types.FetchMessageHeaders:
		return w.handleFetchMessageHeaders(msg)
	case *types.FetchMessageBodyPart:
		return w.handleFetchMessageBodyPart(msg)
	case *types.FetchFullMessages:
		return w.handleFetchFullMessages(msg)
	case *types.FetchMessageFlags:
		return w.handleFetchMessageFlags(msg)
	case *types.DeleteMessages:
		return w.handleDeleteMessages(msg)
	case *types.FlagMessages:
		return w.handleFlagMessages(msg)
	case *types.AnsweredMessages:
		return w.handleAnsweredMessages(msg)
	case *types.CopyMessages:
		return w.handleCopyMessages(msg)
	case *types.MoveMessages:
		return w.handleMoveMessages(msg)
	case *types.AppendMessage:
		return w.handleAppendMessage(msg)
	case *types.SearchDirectory:
		return w.handleSearchDirectory(msg)
	case *types.CheckMail:
		return w.handleCheckMailMessage(msg)
	case *types.ModifyLabels:
		return w.handleModifyLabels(msg)
	}

	return types.ErrUnsupported
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
			w.worker.PostAction(context.TODO(), &types.CheckMail{
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
		systemFlags, keywordFlags := translateImapFlags(msg.Flags)
		w.worker.PostMessage(&types.MessageInfo{
			Info: &models.MessageInfo{
				BodyStructure: translateBodyStructure(msg.BodyStructure),
				Envelope:      translateEnvelope(msg.Envelope),
				Flags:         systemFlags,
				Labels:        keywordFlags,
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

	w.idler.SetClient(nil)
}

func (w *IMAPWorker) stopIdler() error {
	if w.idler.client == nil {
		return nil
	}

	if err := w.idler.Stop(); err != nil {
		w.terminate()
		w.observer.EmitIfNotConnected()
		w.worker.Errorf("idler stopped with error:%v", err)
		return err
	}

	// set connection timeout for calls to imap server
	if w.client != nil {
		w.client.Timeout = w.config.connection_timeout
	}

	return nil
}

func (w *IMAPWorker) startIdler() {
	if w.idler.client == nil {
		return
	}
	// we don't want idle to timeout, so set timeout to zero
	if w.client != nil {
		w.client.Timeout = 0
	}

	w.idler.Start()
}

func (w *IMAPWorker) Run() {
	defer log.PanicHandler()
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

			err := w.handleMessage(msg)

			switch {
			case errors.Is(err, types.ErrNoop):
				// Operation did not have any effect.
				// Do *NOT* send a Done message.
				break
			case errors.Is(err, context.Canceled):
				w.worker.PostMessage(&types.Cancelled{
					Message: types.RespondTo(msg),
				}, nil)
			case errors.Is(err, types.ErrUnsupported):
				w.worker.PostMessage(&types.Unsupported{
					Message: types.RespondTo(msg),
				}, nil)
			case err != nil:
				w.worker.PostMessage(&types.Error{
					Message: types.RespondTo(msg),
					Error:   err,
				}, nil)
			default: // err == nil
				// Operation is finished.
				// Send a Done message.
				w.worker.PostMessage(&types.Done{
					Message: types.RespondTo(msg),
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
