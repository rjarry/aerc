package jmap

import (
	"errors"
	"net/url"
	"time"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib/uidstore"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/handlers"
	"git.sr.ht/~rjarry/aerc/worker/jmap/cache"
	"git.sr.ht/~rjarry/aerc/worker/types"
	"git.sr.ht/~rockorager/go-jmap"
	"git.sr.ht/~rockorager/go-jmap/mail/identity"
	"git.sr.ht/~rockorager/go-jmap/mail/mailbox"
)

func init() {
	handlers.RegisterWorkerFactory("jmap", NewJMAPWorker)
}

var errUnsupported = errors.New("unsupported")

type JMAPWorker struct {
	config struct {
		account    *config.AccountConfig
		endpoint   string
		oauth      bool
		user       *url.Userinfo
		cacheState bool
		cacheBlobs bool
		serverPing time.Duration
		useLabels  bool
		allMail    string
	}

	w         *types.Worker
	client    *jmap.Client
	cache     *cache.JMAPCache
	accountId jmap.ID

	selectedMbox jmap.ID
	dir2mbox     map[string]jmap.ID
	mbox2dir     map[jmap.ID]string
	roles        map[mailbox.Role]jmap.ID
	identities   map[string]*identity.Identity
	uidStore     *uidstore.Store

	changes chan jmap.TypeState
	stop    chan struct{}
}

func NewJMAPWorker(worker *types.Worker) (types.Backend, error) {
	return &JMAPWorker{
		w:          worker,
		uidStore:   uidstore.NewStore(),
		roles:      make(map[mailbox.Role]jmap.ID),
		dir2mbox:   make(map[string]jmap.ID),
		mbox2dir:   make(map[jmap.ID]string),
		identities: make(map[string]*identity.Identity),
		changes:    make(chan jmap.TypeState),
	}, nil
}

func (w *JMAPWorker) addMbox(mbox *mailbox.Mailbox, dir string) {
	w.mbox2dir[mbox.ID] = dir
	w.dir2mbox[dir] = mbox.ID
	w.roles[mbox.Role] = mbox.ID
}

func (w *JMAPWorker) deleteMbox(id jmap.ID) {
	var dir string
	var role mailbox.Role

	delete(w.mbox2dir, id)
	for d, i := range w.dir2mbox {
		if i == id {
			dir = d
			break
		}
	}
	delete(w.dir2mbox, dir)
	for r, i := range w.roles {
		if i == id {
			role = r
			break
		}
	}
	delete(w.roles, role)
}

var capas = models.Capabilities{Sort: true, Thread: false}

func (w *JMAPWorker) Capabilities() *models.Capabilities {
	return &capas
}

func (w *JMAPWorker) PathSeparator() string {
	return "/"
}

func (w *JMAPWorker) handleMessage(msg types.WorkerMessage) error {
	switch msg := msg.(type) {
	case *types.Unsupported:
		// No-op
		break
	case *types.Configure:
		return w.handleConfigure(msg)
	case *types.Connect:
		if w.stop != nil {
			return errors.New("already connected")
		}
		return w.handleConnect(msg)
	case *types.Reconnect:
		if w.stop == nil {
			return errors.New("not connected")
		}
		close(w.stop)
		return w.handleConnect(&types.Connect{Message: msg.Message})
	case *types.Disconnect:
		if w.stop == nil {
			return errors.New("not connected")
		}
		close(w.stop)
		return nil
	case *types.ListDirectories:
		return w.handleListDirectories(msg)
	case *types.OpenDirectory:
		return w.handleOpenDirectory(msg)
	case *types.FetchDirectoryContents:
		return w.handleFetchDirectoryContents(msg)
	case *types.SearchDirectory:
		return w.handleSearchDirectory(msg)
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
	case *types.FlagMessages:
		return w.updateFlags(msg.Uids, msg.Flags, msg.Enable)
	case *types.AnsweredMessages:
		return w.updateFlags(msg.Uids, models.AnsweredFlag, msg.Answered)
	case *types.DeleteMessages:
		return w.moveCopy(msg.Uids, "", true)
	case *types.CopyMessages:
		return w.moveCopy(msg.Uids, msg.Destination, false)
	case *types.MoveMessages:
		return w.moveCopy(msg.Uids, msg.Destination, true)
	case *types.ModifyLabels:
		if w.config.useLabels {
			return w.handleModifyLabels(msg)
		}
	case *types.AppendMessage:
		return w.handleAppendMessage(msg)
	case *types.StartSendingMessage:
		return w.handleStartSend(msg)
	}
	return errUnsupported
}

func (w *JMAPWorker) Run() {
	for {
		select {
		case change := <-w.changes:
			err := w.refresh(change)
			if err != nil {
				w.w.Errorf("refresh: %s", err)
			}
		case msg := <-w.w.Actions:
			msg = w.w.ProcessAction(msg)
			err := w.handleMessage(msg)
			switch {
			case errors.Is(err, errUnsupported):
				w.w.PostMessage(&types.Unsupported{
					Message: types.RespondTo(msg),
				}, nil)
			case err != nil:
				w.w.PostMessage(&types.Error{
					Message: types.RespondTo(msg),
					Error:   err,
				}, nil)
			default:
				w.w.PostMessage(&types.Done{
					Message: types.RespondTo(msg),
				}, nil)
			}
		}
	}
}
