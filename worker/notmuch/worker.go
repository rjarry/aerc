//go:build notmuch
// +build notmuch

package notmuch

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/lib/watchers"
	"git.sr.ht/~rjarry/aerc/lib/xdg"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/handlers"
	"git.sr.ht/~rjarry/aerc/worker/lib"
	notmuch "git.sr.ht/~rjarry/aerc/worker/notmuch/lib"
	"git.sr.ht/~rjarry/aerc/worker/types"
	"github.com/emersion/go-maildir"
)

func init() {
	handlers.RegisterWorkerFactory("notmuch", NewWorker)
}

type worker struct {
	w                   *types.Worker
	nmStateChange       chan bool
	query               string
	currentQueryName    string
	queryMapOrder       []string
	nameQueryMap        map[string]string
	dynamicNameQueryMap map[string]string
	store               *lib.MaildirStore
	maildirAccountPath  string
	db                  *notmuch.DB
	setupErr            error
	watcher             watchers.FSWatcher
	watcherDebounce     *time.Timer
	capabilities        *models.Capabilities
	headers             []string
	headersExclude      []string
	state               uint64
	mfs                 types.MultiFileStrategy
}

// NewWorker creates a new notmuch worker with the provided worker.
func NewWorker(w *types.Worker) (types.Backend, error) {
	events := make(chan bool, 20)
	watcher, err := watchers.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("could not create file system watcher: %w", err)
	}
	return &worker{
		w:             w,
		nmStateChange: events,
		watcher:       watcher,
		capabilities: &models.Capabilities{
			Sort:   true,
			Thread: true,
		},
		dynamicNameQueryMap: make(map[string]string),
	}, nil
}

// Run starts the worker's message handling loop.
func (w *worker) Run() {
	defer log.PanicHandler()
	for {
		select {
		case action := <-w.w.Actions():
			msg := w.w.ProcessAction(action)
			err := w.handleMessage(msg)
			switch {
			case errors.Is(err, types.ErrNoop):
				// Operation did not have any effect.
				// Do *NOT* send a Done message.
				break
			case errors.Is(err, context.Canceled):
				w.w.PostMessage(&types.Cancelled{
					Message: types.RespondTo(msg),
				}, nil)
			case errors.Is(err, types.ErrUnsupported):
				w.w.PostMessage(&types.Unsupported{
					Message: types.RespondTo(msg),
				}, nil)
			case err != nil:
				w.w.PostMessage(&types.Error{
					Message: types.RespondTo(msg),
					Error:   err,
				}, nil)
			default: // err == nil
				// Operation is finished.
				// Send a Done message.
				w.w.PostMessage(&types.Done{
					Message: types.RespondTo(msg),
				}, nil)
			}
		case <-w.nmStateChange:
			err := w.handleNotmuchEvent()
			if err != nil {
				w.w.Errorf("notmuch event failure: %v", err)
			}
		case <-w.watcher.Events():
			if w.watcherDebounce != nil {
				w.watcherDebounce.Stop()
			}
			// Debounce FS changes
			w.watcherDebounce = time.AfterFunc(50*time.Millisecond, func() {
				defer log.PanicHandler()
				w.nmStateChange <- true
			})
		}
	}
}

func (w *worker) Capabilities() *models.Capabilities {
	return w.capabilities
}

func (w *worker) PathSeparator() string {
	// make it configurable?
	// <rockorager> You can use those in query maps to force a tree
	// <rockorager> Might be nice to be configurable? I see some notmuch people namespace with "::"
	return "/"
}

func (w *worker) handleMessage(msg types.WorkerMessage) error {
	if w.setupErr != nil {
		// only configure can recover from a config error, bail for everything else
		_, isConfigure := msg.(*types.Configure)
		if !isConfigure {
			return w.setupErr
		}
	}
	if w.db != nil {
		err := w.db.Connect()
		if err != nil {
			return err
		}
		defer w.db.Close()
	}

	switch msg := msg.(type) {
	case *types.Unsupported:
		// No-op
	case *types.Configure:
		return w.handleConfigure(msg)
	case *types.Connect:
		return w.handleConnect(msg)
	case *types.ListDirectories:
		return w.handleListDirectories(msg)
	case *types.OpenDirectory:
		return w.handleOpenDirectory(msg)
	case *types.FetchDirectoryContents:
		return w.emitDirectoryContents(msg)
	case *types.FetchDirectoryThreaded:
		return w.emitDirectoryThreaded(msg)
	case *types.FetchMessageHeaders:
		return w.handleFetchMessageHeaders(msg)
	case *types.FetchMessageBodyPart:
		return w.handleFetchMessageBodyPart(msg)
	case *types.FetchFullMessages:
		return w.handleFetchFullMessages(msg)
	case *types.FlagMessages:
		return w.handleFlagMessages(msg)
	case *types.AnsweredMessages:
		return w.handleAnsweredMessages(msg)
	case *types.ForwardedMessages:
		return w.handleForwardedMessages(msg)
	case *types.SearchDirectory:
		return w.handleSearchDirectory(msg)
	case *types.ModifyLabels:
		return w.handleModifyLabels(msg)
	case *types.CheckMail:
		go w.handleCheckMail(msg)
		return nil
	case *types.DeleteMessages:
		return w.handleDeleteMessages(msg)
	case *types.CopyMessages:
		return w.handleCopyMessages(msg)
	case *types.MoveMessages:
		return w.handleMoveMessages(msg)
	case *types.AppendMessage:
		return w.handleAppendMessage(msg)
	case *types.CreateDirectory:
		return w.handleCreateDirectory(msg)
	case *types.RemoveDirectory:
		return w.handleRemoveDirectory(msg)
	}
	return types.ErrUnsupported
}

func (w *worker) handleConfigure(msg *types.Configure) error {
	var err error
	defer func() {
		if err == nil {
			w.setupErr = nil
			return
		}
		w.setupErr = fmt.Errorf("notmuch: %w", err)
	}()

	u, err := url.Parse(msg.Config.Source)
	if err != nil {
		w.w.Errorf("error configuring notmuch worker: %v", err)
		return err
	}
	home := xdg.ExpandHome(u.Hostname())
	pathToDB := filepath.Join(home, u.Path)
	err = w.loadQueryMap(msg.Config)
	if err != nil {
		return fmt.Errorf("could not load query map configuration: %w", err)
	}
	excludedTags := w.loadExcludeTags(msg.Config)
	w.db = notmuch.NewDB(pathToDB, excludedTags)

	val, ok := msg.Config.Params["maildir-store"]
	if ok {
		path := xdg.ExpandHome(val)
		w.maildirAccountPath = msg.Config.Params["maildir-account-path"]

		path = filepath.Join(path, w.maildirAccountPath)
		store, err := lib.NewMaildirStore(path, false)
		if err != nil {
			return fmt.Errorf("Cannot initialize maildir store: %w", err)
		}
		w.store = store
	}
	w.headers = msg.Config.Headers
	w.headersExclude = msg.Config.HeadersExclude

	mfs := msg.Config.Params["multi-file-strategy"]
	if mfs != "" {
		w.mfs, ok = types.StrToStrategy[mfs]
		if !ok {
			return fmt.Errorf("invalid multi-file strategy %s", mfs)
		}
	} else {
		w.mfs = types.Refuse
	}

	return nil
}

func (w *worker) handleConnect(msg *types.Connect) error {
	w.emitLabelList()
	// Get initial db state
	w.state = w.db.State()
	// Watch all the files in the xapian folder for changes. We'll debounce
	// changes, so catching multiple is ok
	var dbPath string
	path := filepath.Join(w.db.Path(), ".notmuch", "xapian")
	_, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			dbPath = filepath.Join(w.db.Path(), "xapian")
		} else {
			return fmt.Errorf("error locating notmuch db: %w", err)
		}
	} else {
		dbPath = path
	}

	err = w.watcher.Configure(dbPath)
	log.Tracef("Configuring watcher for path: %v", dbPath)
	if err != nil {
		return fmt.Errorf("error configuring watcher: %w", err)
	}
	return nil
}

func (w *worker) handleListDirectories(msg *types.ListDirectories) error {
	if w.store != nil {
		folders, err := w.store.FolderMap()
		if err != nil {
			w.w.Errorf("failed listing directories: %v", err)
			return err
		}
		for name := range folders {
			w.w.PostMessage(&types.Directory{
				Message: types.RespondTo(msg),
				Dir: &models.Directory{
					Name: name,
				},
			}, nil)
		}
	}

	for _, name := range w.queryMapOrder {
		w.w.PostMessage(&types.Directory{
			Message: types.RespondTo(msg),
			Dir: &models.Directory{
				Name: name,
				Role: models.QueryRole,
			},
		}, nil)
	}

	for name := range w.dynamicNameQueryMap {
		w.w.PostMessage(&types.Directory{
			Message: types.RespondTo(msg),
			Dir: &models.Directory{
				Name: name,
				Role: models.QueryRole,
			},
		}, nil)
	}

	// Update dir counts when listing directories
	return w.updateDirCounts()
}

func (w *worker) getDirectoryInfo(name string, query string) *models.DirectoryInfo {
	dirInfo := &models.DirectoryInfo{
		Name: name,
		// total messages
		Exists: 0,
		// new messages since mailbox was last opened
		Recent: 0,
		// total unread
		Unseen: 0,
	}

	count, err := w.db.QueryCountMessages(query)
	if err != nil {
		return dirInfo
	}
	dirInfo.Exists = count.Exists
	dirInfo.Unseen = count.Unread

	return dirInfo
}

func (w *worker) handleOpenDirectory(msg *types.OpenDirectory) error {
	if msg.Context().Err() != nil {
		return context.Canceled
	}
	w.w.Tracef("opening %s with query %s", msg.Directory, msg.Query)

	var exists bool
	q := ""
	if w.store != nil {
		folders, _ := w.store.FolderMap()
		var dir maildir.Dir
		dir, exists = folders[msg.Directory]
		if exists {
			folder := filepath.Join(w.maildirAccountPath, msg.Directory)
			q = fmt.Sprintf("folder:%s", strconv.Quote(folder))
			if err := w.processNewMaildirFiles(string(dir)); err != nil {
				return err
			}
		}
	}
	if q == "" {
		q, exists = w.nameQueryMap[msg.Directory]
		if !exists {
			q, exists = w.dynamicNameQueryMap[msg.Directory]
		}
	}
	if !exists || msg.Force {
		q = msg.Query
		if q == "" {
			q = msg.Directory
		}
		w.dynamicNameQueryMap[msg.Directory] = q
		w.w.PostMessage(&types.Directory{
			Message: types.RespondTo(msg),
			Dir: &models.Directory{
				Name: msg.Directory,
				Role: models.QueryRole,
			},
		}, nil)
	} else if msg.Query != "" && msg.Query != q {
		return errors.New("cannot use existing folder name for new query")
	}
	w.query = q
	w.currentQueryName = msg.Directory

	w.w.PostMessage(&types.DirectoryInfo{
		Info:    w.getDirectoryInfo(msg.Directory, w.query),
		Message: types.RespondTo(msg),
	}, nil)
	return nil
}

func (w *worker) handleFetchMessageHeaders(
	msg *types.FetchMessageHeaders,
) error {
	for _, uid := range msg.Uids {
		m, err := w.msgFromUid(uid)
		if err != nil {
			w.w.Errorf("could not get message: %v", err)
			w.emitMessageInfoError(msg, msg.Directory, uid, err)
			continue
		}
		err = w.emitMessageInfo(m, msg.Directory, msg)
		if err != nil {
			w.w.Errorf("could not emit message info: %v", err)
			w.emitMessageInfoError(msg, msg.Directory, uid, err)
			continue
		}
	}
	return nil
}

func (w *worker) uidsFromQuery(ctx context.Context, query string) ([]models.UID, error) {
	msgIDs, err := w.db.MsgIDsFromQuery(ctx, query)
	if err != nil {
		return nil, err
	}
	var uids []models.UID
	for _, id := range msgIDs {
		uids = append(uids, models.UID(id))
	}
	return uids, nil
}

func (w *worker) msgFromUid(uid models.UID) (*Message, error) {
	msg := &Message{
		key: string(uid),
		uid: uid,
		db:  w.db,
	}
	return msg, nil
}

func (w *worker) handleFetchMessageBodyPart(
	msg *types.FetchMessageBodyPart,
) error {
	m, err := w.msgFromUid(msg.Uid)
	if err != nil {
		w.w.Errorf("could not get message %d: %v", msg.Uid, err)
		return err
	}
	r, err := m.NewBodyPartReader(msg.Part)
	if err != nil {
		w.w.Errorf(
			"could not get body part reader for message=%d, parts=%#v: %w",
			msg.Uid, msg.Part, err)
		return err
	}
	w.w.PostMessage(&types.MessageBodyPart{
		Message: types.RespondTo(msg),
		Part: &models.MessageBodyPart{
			Reader: r,
			Uid:    msg.Uid,
		},
	}, nil)
	return nil
}

func (w *worker) handleFetchFullMessages(msg *types.FetchFullMessages) error {
	for _, uid := range msg.Uids {
		m, err := w.msgFromUid(uid)
		if err != nil {
			w.w.Errorf("could not get message %d: %v", uid, err)
			return err
		}
		r, err := m.NewReader()
		if err != nil {
			w.w.Errorf("could not get message reader: %v", err)
			return err
		}
		defer r.Close()
		b, err := io.ReadAll(r)
		if err != nil {
			return err
		}
		w.w.PostMessage(&types.FullMessage{
			Message: types.RespondTo(msg),
			Content: &models.FullMessage{
				Uid:    uid,
				Reader: bytes.NewReader(b),
			},
		}, nil)
	}
	return nil
}

func (w *worker) handleAnsweredMessages(msg *types.AnsweredMessages) error {
	for _, uid := range msg.Uids {
		m, err := w.msgFromUid(uid)
		if err != nil {
			w.w.Errorf("could not get message: %v", err)
			return err
		}
		if err := m.MarkAnswered(msg.Answered); err != nil {
			w.w.Errorf("could not mark message as answered: %v", err)
			return err
		}
	}
	return nil
}

func (w *worker) handleForwardedMessages(msg *types.ForwardedMessages) error {
	for _, uid := range msg.Uids {
		m, err := w.msgFromUid(uid)
		if err != nil {
			w.w.Errorf("could not get message: %v", err)
			return err
		}
		if err := m.MarkForwarded(msg.Forwarded); err != nil {
			w.w.Errorf("could not mark message as forwarded: %v", err)
			return err
		}
	}
	return nil
}

func (w *worker) handleFlagMessages(msg *types.FlagMessages) error {
	for _, uid := range msg.Uids {
		m, err := w.msgFromUid(uid)
		if err != nil {
			w.w.Errorf("could not get message: %v", err)
			return err
		}
		if err := m.SetFlag(msg.Flags, msg.Enable); err != nil {
			w.w.Errorf("could not set flag %v as %t for message: %v",
				msg.Flags, msg.Enable, err)
			return err
		}
	}
	return nil
}

func (w *worker) handleSearchDirectory(msg *types.SearchDirectory) error {
	search := notmuch.AndQueries(w.query, translate(msg.Criteria))
	log.Debugf("search query: '%s'", search)
	uids, err := w.uidsFromQuery(msg.Context(), search)
	if err != nil {
		return err
	}
	w.w.PostMessage(&types.SearchResults{
		Message:   types.RespondTo(msg),
		Directory: w.currentQueryName,
		Criteria:  msg.Criteria,
		Uids:      uids,
	}, nil)
	return nil
}

func (w *worker) handleModifyLabels(msg *types.ModifyLabels) error {
	for _, uid := range msg.Uids {
		m, err := w.msgFromUid(uid)
		if err != nil {
			return fmt.Errorf("could not get message from uid %s: %w", uid, err)
		}
		err = m.ModifyTags(msg.Add, msg.Remove, msg.Toggle)
		if err != nil {
			return fmt.Errorf("could not modify message tags: %w", err)
		}
	}
	return nil
}

func (w *worker) loadQueryMap(acctConfig *config.AccountConfig) error {
	raw, ok := acctConfig.Params["query-map"]
	if !ok {
		// nothing to do
		return nil
	}
	file := xdg.ExpandHome(raw)
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()
	w.nameQueryMap, w.queryMapOrder, err = lib.ParseFolderMap(f)
	return err
}

func (w *worker) loadExcludeTags(
	acctConfig *config.AccountConfig,
) []string {
	raw, ok := acctConfig.Params["exclude-tags"]
	if !ok {
		// nothing to do
		return nil
	}
	excludedTags := strings.Split(raw, ",")
	for idx, tag := range excludedTags {
		excludedTags[idx] = strings.Trim(tag, " ")
	}
	return excludedTags
}

func (w *worker) emitDirectoryContents(msg *types.FetchDirectoryContents) error {
	query := notmuch.AndQueries(w.query, translate(msg.Filter))
	log.Debugf("filter query: '%s'", query)
	uids, err := w.uidsFromQuery(msg.Context(), query)
	if err != nil {
		return fmt.Errorf("could not fetch uids: %w", err)
	}
	sortedUids, err := w.sort(uids, msg.SortCriteria, msg.Directory)
	if err != nil {
		w.w.Errorf("error sorting directory: %v", err)
		return err
	}
	w.w.PostMessage(&types.DirectoryContents{
		Message:   types.RespondTo(msg),
		Directory: msg.Directory,
		Filter:    msg.Filter,
		Uids:      sortedUids,
	}, nil)
	return nil
}

func (w *worker) emitDirectoryThreaded(msg *types.FetchDirectoryThreaded) error {
	query := notmuch.AndQueries(w.query, translate(msg.Filter))
	log.Debugf("filter query: '%s'", query)
	threads, err := w.db.ThreadsFromQuery(msg.Context(), query, msg.ThreadContext)
	if err != nil {
		return err
	}
	w.w.PostMessage(&types.DirectoryThreaded{
		Message:   types.RespondTo(msg),
		Directory: msg.Directory,
		Filter:    msg.Filter,
		Threads:   threads,
	}, nil)
	return nil
}

func (w *worker) emitMessageInfoError(msg types.WorkerMessage, dir string, uid models.UID, err error) {
	w.w.PostMessage(&types.MessageInfo{
		Message: types.RespondTo(msg),
		Info: &models.MessageInfo{
			Envelope:  &models.Envelope{},
			Flags:     models.SeenFlag,
			Directory: dir,
			Uid:       uid,
			Error:     err,
		},
	}, nil)
}

func (w *worker) emitMessageInfo(m *Message, dir string,
	parent types.WorkerMessage,
) error {
	info, err := m.MessageInfo(dir)
	if err != nil {
		return fmt.Errorf("could not get MessageInfo: %w", err)
	}
	switch {
	case len(w.headersExclude) > 0:
		info.RFC822Headers = lib.LimitHeaders(info.RFC822Headers, w.headersExclude, true)
	case len(w.headers) > 0:
		info.RFC822Headers = lib.LimitHeaders(info.RFC822Headers, w.headers, false)
	}
	switch parent {
	case nil:
		w.w.PostMessage(&types.MessageInfo{
			Info: info,
		}, nil)
	default:
		w.w.PostMessage(&types.MessageInfo{
			Message: types.RespondTo(parent),
			Info:    info,
		}, nil)
	}
	return nil
}

func (w *worker) emitLabelList() {
	tags := w.db.ListTags()
	w.w.PostMessage(&types.LabelList{Labels: tags}, nil)
}

func (w *worker) sort(
	uids []models.UID, criteria []*types.SortCriterion, dir string,
) ([]models.UID, error) {
	if len(criteria) == 0 {
		return uids, nil
	}
	var msgInfos []*models.MessageInfo
	for _, uid := range uids {
		m, err := w.msgFromUid(uid)
		if err != nil {
			w.w.Errorf("could not get message: %v", err)
			continue
		}
		info, err := m.MessageInfo(dir)
		if err != nil {
			w.w.Errorf("could not get message info: %v", err)
			continue
		}
		msgInfos = append(msgInfos, info)
	}
	sortedUids, err := lib.Sort(msgInfos, criteria)
	if err != nil {
		w.w.Errorf("could not sort the messages: %v", err)
		return nil, err
	}
	return sortedUids, nil
}

func (w *worker) handleCheckMail(msg *types.CheckMail) {
	defer log.PanicHandler()
	if msg.Command == "" {
		w.w.PostMessage(&types.Error{
			Message: types.RespondTo(msg),
			Error:   fmt.Errorf("(%s) checkmail: no command specified", msg.Account()),
		}, nil)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), msg.Timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", "-c", msg.Command)
	err := cmd.Run()
	switch {
	case ctx.Err() != nil:
		w.w.PostMessage(&types.Error{
			Message: types.RespondTo(msg),
			Error:   fmt.Errorf("(%s) checkmail: timed out", msg.Account()),
		}, nil)
	case err != nil:
		w.w.PostMessage(&types.Error{
			Message: types.RespondTo(msg),
			Error:   fmt.Errorf("(%s) checkmail: error running command: %w", msg.Account(), err),
		}, nil)
	default:
		w.w.PostMessage(&types.Done{
			Message: types.RespondTo(msg),
		}, nil)
	}
}

// folderDir returns the maildir.Dir for the given folder name. If name is
// empty, returns the currently selected folder.
func (w *worker) folderDir(folders map[string]maildir.Dir, name string) maildir.Dir {
	if name != "" {
		if dir, ok := folders[name]; ok {
			return dir
		}
	}
	return folders[w.currentQueryName]
}

func (w *worker) handleDeleteMessages(msg *types.DeleteMessages) error {
	if w.store == nil {
		return types.ErrUnsupported
	}

	var deleted []models.UID

	folders, _ := w.store.FolderMap()
	curDir := w.folderDir(folders, msg.Directory)

	mfs := w.mfs
	if msg.MultiFileStrategy != nil {
		mfs = *msg.MultiFileStrategy
	}

	for _, uid := range msg.Uids {
		m, err := w.msgFromUid(uid)
		if err != nil {
			w.w.Errorf("could not get message: %v", err)
			return err
		}
		if err := m.Remove(curDir, mfs); err != nil {
			w.w.Errorf("could not remove message: %v", err)
			return err
		}
		deleted = append(deleted, uid)
	}
	if len(deleted) > 0 {
		w.w.PostMessage(&types.MessagesDeleted{
			Message:   types.RespondTo(msg),
			Directory: msg.Directory,
			Uids:      deleted,
		}, nil)
	}
	return nil
}

func (w *worker) handleCopyMessages(msg *types.CopyMessages) error {
	if w.store == nil {
		return types.ErrUnsupported
	}

	// Only allow file to be copied to a maildir folder
	folders, _ := w.store.FolderMap()
	dest, ok := folders[msg.Destination]
	if !ok {
		return fmt.Errorf("Can only copy file to a maildir folder")
	}

	curDir := w.folderDir(folders, msg.Source)

	mfs := w.mfs
	if msg.MultiFileStrategy != nil {
		mfs = *msg.MultiFileStrategy
	}

	for _, uid := range msg.Uids {
		m, err := w.msgFromUid(uid)
		if err != nil {
			w.w.Errorf("could not get message: %v", err)
			return err
		}
		if err := m.Copy(curDir, dest, mfs); err != nil {
			w.w.Errorf("could not copy message: %v", err)
			return err
		}
	}
	w.w.PostMessage(&types.MessagesCopied{
		Message:     types.RespondTo(msg),
		Destination: msg.Destination,
		Uids:        msg.Uids,
	}, nil)
	return nil
}

func (w *worker) handleMoveMessages(msg *types.MoveMessages) error {
	if w.store == nil {
		return types.ErrUnsupported
	}

	folders, _ := w.store.FolderMap()

	// Only allow file to be moved to a maildir folder
	dest, ok := folders[msg.Destination]
	if !ok {
		return fmt.Errorf("Can only move file to a maildir folder")
	}

	curDir := w.folderDir(folders, msg.Source)

	mfs := w.mfs
	if msg.MultiFileStrategy != nil {
		mfs = *msg.MultiFileStrategy
	}

	for _, uid := range msg.Uids {
		m, err := w.msgFromUid(uid)
		if err != nil {
			w.w.Errorf("could not get message: %v", err)
			return err
		}
		if err := m.Move(curDir, dest, mfs); err != nil {
			w.w.Errorf("could not move message: %v", err)
			return err
		}
	}
	w.w.PostMessage(&types.MessagesDeleted{
		Message:   types.RespondTo(msg),
		Directory: msg.Source,
		Uids:      msg.Uids,
	}, nil)
	return nil
}

func (w *worker) handleAppendMessage(msg *types.AppendMessage) error {
	if w.store == nil {
		return types.ErrUnsupported
	}

	// Only allow file to be created in a maildir folder
	// since we are the "master" maildir process, we can modify the maildir directly
	folders, _ := w.store.FolderMap()
	dest, ok := folders[msg.Destination]
	if !ok {
		return fmt.Errorf("Can only create file in a maildir folder")
	}
	newMsg, writer, err := dest.Create(lib.ToMaildirFlags(msg.Flags))
	if err != nil {
		w.w.Errorf("could not create message at %s: %v", msg.Destination, err)
		return err
	}
	filename := newMsg.Filename()
	if _, err := io.Copy(writer, msg.Reader); err != nil {
		w.w.Errorf("could not write message to destination: %v", err)
		writer.Close()
		os.Remove(filename)
		return err
	}
	writer.Close()
	id, err := w.db.IndexFile(filename)
	if err != nil {
		return err
	}

	err = w.addFlags(id, msg.Flags)
	if err != nil {
		return err
	}

	w.w.PostMessage(&types.DirectoryInfo{
		Info: w.getDirectoryInfo(w.currentQueryName, w.query),
	}, nil)
	return nil
}

func (w *worker) handleCreateDirectory(msg *types.CreateDirectory) error {
	if w.store == nil {
		return types.ErrUnsupported
	}

	dir := w.store.Dir(msg.Directory)
	if err := dir.Init(); err != nil {
		w.w.Errorf("could not create directory %s: %v",
			msg.Directory, err)
		return err
	}
	return nil
}

func (w *worker) handleRemoveDirectory(msg *types.RemoveDirectory) error {
	_, inQueryMap := w.nameQueryMap[msg.Directory]
	if inQueryMap {
		return types.ErrUnsupported
	}

	if _, ok := w.dynamicNameQueryMap[msg.Directory]; ok {
		delete(w.dynamicNameQueryMap, msg.Directory)
		return nil
	}

	if w.store == nil {
		return nil
	}

	dir := w.store.Dir(msg.Directory)
	if err := os.RemoveAll(string(dir)); err != nil {
		w.w.Errorf("could not remove directory %s: %v",
			msg.Directory, err)
		return err
	}
	return nil
}

// This is a hack that calls MsgModifyTags with an empty list of tags to
// apply on new messages causing notmuch to rename files and effectively
// move them into the cur/ dir.
func (w *worker) processNewMaildirFiles(dir string) error {
	f, err := os.Open(filepath.Join(dir, "new"))
	if err != nil {
		return err
	}
	defer f.Close()
	names, err := f.Readdirnames(0)
	if err != nil {
		return err
	}

	for _, n := range names {
		if n[0] == '.' {
			continue
		}

		key, err := w.db.MsgIDFromFilename(filepath.Join(dir, "new", n))
		if err != nil {
			// Message is not yet indexed, leave it alone
			continue
		}
		// Force message to move from new/ to cur/
		err = w.db.MsgModifyTags(key, nil, nil, nil)
		if err != nil {
			w.w.Errorf("MsgModifyTags failed: %v", err)
		}
	}

	return nil
}

func (w *worker) addFlags(id string, flags models.Flags) error {
	addTags := []string{}
	removeTags := []string{}
	for flag, tag := range flagToTag {
		if !flags.Has(flag) {
			continue
		}

		if flagToInvert[flag] {
			removeTags = append(removeTags, tag)
		} else {
			addTags = append(addTags, tag)
		}
	}

	return w.db.MsgModifyTags(id, addTags, removeTags, nil)
}
