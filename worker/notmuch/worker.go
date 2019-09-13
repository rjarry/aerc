//+build notmuch

package notmuch

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"git.sr.ht/~sircmpwn/aerc/config"
	"git.sr.ht/~sircmpwn/aerc/lib/uidstore"
	"git.sr.ht/~sircmpwn/aerc/models"
	"git.sr.ht/~sircmpwn/aerc/worker/handlers"
	notmuch "git.sr.ht/~sircmpwn/aerc/worker/notmuch/lib"
	"git.sr.ht/~sircmpwn/aerc/worker/types"
	"github.com/mitchellh/go-homedir"
)

func init() {
	handlers.RegisterWorkerFactory("notmuch", NewWorker)
}

var errUnsupported = fmt.Errorf("unsupported command")

type worker struct {
	w            *types.Worker
	query        string
	uidStore     *uidstore.Store
	nameQueryMap map[string]string
	db           *notmuch.DB
}

// NewWorker creates a new maildir worker with the provided worker.
func NewWorker(w *types.Worker) (types.Backend, error) {
	return &worker{w: w}, nil
}

// Run starts the worker's message handling loop.
func (w *worker) Run() {
	for {
		action := <-w.w.Actions
		msg := w.w.ProcessAction(action)
		if err := w.handleMessage(msg); err == errUnsupported {
			w.w.PostMessage(&types.Unsupported{
				Message: types.RespondTo(msg),
			}, nil)
		} else if err != nil {
			w.w.PostMessage(&types.Error{
				Message: types.RespondTo(msg),
				Error:   err,
			}, nil)
		}
	}
}

func (w *worker) done(msg types.WorkerMessage) {
	w.w.PostMessage(&types.Done{types.RespondTo(msg)}, nil)
}

func (w *worker) err(msg types.WorkerMessage, err error) {
	w.w.PostMessage(&types.Error{
		Message: types.RespondTo(msg),
		Error:   err,
	}, nil)
}
func (w *worker) handleMessage(msg types.WorkerMessage) error {
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
		return w.handleFetchDirectoryContents(msg)
	case *types.FetchMessageHeaders:
		return w.handleFetchMessageHeaders(msg)
	case *types.FetchMessageBodyPart:
		return w.handleFetchMessageBodyPart(msg)
	case *types.FetchFullMessages:
		return w.handleFetchFullMessages(msg)
	case *types.ReadMessages:
		return w.handleReadMessages(msg)
	case *types.SearchDirectory:
		return w.handleSearchDirectory(msg)
	case *types.ModifyLabels:
		return w.handleModifyLabels(msg)

		// not implemented, they are generally not used
		// in a notmuch based workflow
		// case *types.DeleteMessages:
		// case *types.CopyMessages:
		// 	return w.handleCopyMessages(msg)
		// case *types.AppendMessage:
		// 	return w.handleAppendMessage(msg)
		// case *types.CreateDirectory:
		// 	return w.handleCreateDirectory(msg)
	}
	return errUnsupported
}

func (w *worker) handleConfigure(msg *types.Configure) error {
	u, err := url.Parse(msg.Config.Source)
	if err != nil {
		w.w.Logger.Printf("error configuring notmuch worker: %v", err)
		return err
	}
	home, err := homedir.Expand(u.Hostname())
	if err != nil {
		return fmt.Errorf("could not resolve home directory: %v", err)
	}
	pathToDB := filepath.Join(home, u.Path)
	w.uidStore = uidstore.NewStore()
	if err = w.loadQueryMap(msg.Config); err != nil {
		return fmt.Errorf("could not load query map: %v", err)
	}
	excludedTags := w.loadExcludeTags(msg.Config)
	w.db = notmuch.NewDB(pathToDB, excludedTags, w.w.Logger)
	return nil
}

func (w *worker) handleConnect(msg *types.Connect) error {
	err := w.db.Connect()
	if err != nil {
		return err
	}
	w.done(msg)
	return nil
}

func (w *worker) handleListDirectories(msg *types.ListDirectories) error {
	for name := range w.nameQueryMap {
		w.w.PostMessage(&types.Directory{
			Message: types.RespondTo(msg),
			Dir: &models.Directory{
				Name:       name,
				Attributes: []string{},
			},
		}, nil)
	}
	w.done(msg)
	return nil
}

func (w *worker) handleOpenDirectory(msg *types.OpenDirectory) error {
	w.w.Logger.Printf("opening %s", msg.Directory)
	// try the friendly name first, if that fails assume it's a query
	q, ok := w.nameQueryMap[msg.Directory]
	if !ok {
		q = msg.Directory
	}
	w.query = q
	count, err := w.db.QueryCountMessages(w.query)
	if err != nil {
		return err
	}
	info := &types.DirectoryInfo{
		Info: &models.DirectoryInfo{
			Name:     msg.Directory,
			Flags:    []string{},
			ReadOnly: false,
			// total messages
			Exists: count.Exists,
			// new messages since mailbox was last opened
			Recent: 0,
			// total unread
			Unseen: count.Unread,
		},
	}
	//TODO: why does this need to be sent twice??
	w.w.PostMessage(info, nil)
	w.w.PostMessage(info, nil)
	w.done(msg)
	return nil
}

func (w *worker) handleFetchDirectoryContents(
	msg *types.FetchDirectoryContents) error {
	err := w.emitDirectoryContents(msg)
	if err != nil {
		return err
	}
	w.done(msg)
	return nil
}

func (w *worker) handleFetchMessageHeaders(
	msg *types.FetchMessageHeaders) error {
	for _, uid := range msg.Uids {
		m, err := w.msgFromUid(uid)
		if err != nil {
			w.w.Logger.Printf("could not get message: %v", err)
			w.err(msg, err)
			continue
		}
		err = w.emitMessageInfo(m, msg)
		if err != nil {
			w.w.Logger.Printf(err.Error())
			w.err(msg, err)
			continue
		}
	}
	w.done(msg)
	return nil
}

func (w *worker) uidsFromQuery(query string) ([]uint32, error) {
	msgIDs, err := w.db.MsgIDsFromQuery(query)
	if err != nil {
		return nil, err
	}
	var uids []uint32
	for _, id := range msgIDs {
		uid := w.uidStore.GetOrInsert(id)
		uids = append(uids, uid)

	}
	return uids, nil
}

func (w *worker) msgFromUid(uid uint32) (*Message, error) {
	key, ok := w.uidStore.GetKey(uid)
	if !ok {
		return nil, fmt.Errorf("Invalid uid: %v", uid)
	}
	msg := &Message{
		key: key,
		uid: uid,
		db:  w.db,
	}
	return msg, nil
}

func (w *worker) handleFetchMessageBodyPart(
	msg *types.FetchMessageBodyPart) error {

	m, err := w.msgFromUid(msg.Uid)
	if err != nil {
		w.w.Logger.Printf("could not get message %d: %v", msg.Uid, err)
		return err
	}
	r, err := m.NewBodyPartReader(msg.Part)
	if err != nil {
		w.w.Logger.Printf(
			"could not get body part reader for message=%d, parts=%#v: %v",
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

	if err := m.MarkRead(true); err != nil {
		w.w.Logger.Printf("could not mark message as read: %v", err)
		return err
	}

	// send updated flags to ui
	err = w.emitMessageInfo(m, msg)
	if err != nil {
		w.w.Logger.Printf(err.Error())
		w.err(msg, err)
	}
	w.done(msg)
	return nil
}

func (w *worker) handleFetchFullMessages(msg *types.FetchFullMessages) error {
	for _, uid := range msg.Uids {
		m, err := w.msgFromUid(uid)
		if err != nil {
			w.w.Logger.Printf("could not get message %d: %v", uid, err)
			return err
		}
		r, err := m.NewReader()
		if err != nil {
			w.w.Logger.Printf("could not get message reader: %v", err)
			return err
		}
		w.w.PostMessage(&types.FullMessage{
			Message: types.RespondTo(msg),
			Content: &models.FullMessage{
				Uid:    uid,
				Reader: r,
			},
		}, nil)
	}
	w.done(msg)
	return nil
}

func (w *worker) handleReadMessages(msg *types.ReadMessages) error {
	for _, uid := range msg.Uids {
		m, err := w.msgFromUid(uid)
		if err != nil {
			w.w.Logger.Printf("could not get message: %v", err)
			w.err(msg, err)
			continue
		}
		if err := m.MarkRead(msg.Read); err != nil {
			w.w.Logger.Printf("could not mark message as read: %v", err)
			w.err(msg, err)
			continue
		}
		err = w.emitMessageInfo(m, msg)
		if err != nil {
			w.w.Logger.Printf(err.Error())
			w.err(msg, err)
			continue
		}
	}
	w.done(msg)
	return nil
}

func (w *worker) handleSearchDirectory(msg *types.SearchDirectory) error {
	// the first item is the command (search / filter)
	s := strings.Join(msg.Argv[1:], " ")
	// we only want to search in the current query, so merge the two together
	search := fmt.Sprintf("(%v) and (%v)", w.query, s)
	uids, err := w.uidsFromQuery(search)
	if err != nil {
		return err
	}
	w.w.PostMessage(&types.SearchResults{
		Message: types.RespondTo(msg),
		Uids:    uids,
	}, nil)
	return nil
}

func (w *worker) handleModifyLabels(msg *types.ModifyLabels) error {
	for _, uid := range msg.Uids {
		m, err := w.msgFromUid(uid)
		if err != nil {
			return fmt.Errorf("could not get message from uid %v: %v", uid, err)
		}
		err = m.ModifyTags(msg.Add, msg.Remove)
		if err != nil {
			return fmt.Errorf("could not modify message tags: %v", err)
		}
		err = w.emitMessageInfo(m, msg)
		if err != nil {
			return err
		}
	}
	// tags changed, most probably some messages shifted to other folders
	// so we need to re-enumerate the query content
	err := w.emitDirectoryContents(msg)
	if err != nil {
		return err
	}
	w.done(msg)
	return nil
}

func (w *worker) loadQueryMap(acctConfig *config.AccountConfig) error {
	raw, ok := acctConfig.Params["query-map"]
	if !ok {
		// nothing to do
		return nil
	}
	file, err := homedir.Expand(raw)
	if err != nil {
		return err
	}
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()
	w.nameQueryMap = make(map[string]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		split := strings.SplitN(line, "=", 2)
		if len(split) != 2 {
			return fmt.Errorf("invalid line %q, want name=query", line)
		}
		w.nameQueryMap[split[0]] = split[1]
	}
	return nil
}

func (w *worker) loadExcludeTags(
	acctConfig *config.AccountConfig) []string {
	raw, ok := acctConfig.Params["exclude-tags"]
	if !ok {
		// nothing to do
		return nil
	}
	excludedTags := strings.Split(raw, ",")
	return excludedTags
}

func (w *worker) emitDirectoryContents(parent types.WorkerMessage) error {
	uids, err := w.uidsFromQuery(w.query)
	if err != nil {
		return fmt.Errorf("could not fetch uids: %v", err)
	}
	w.w.PostMessage(&types.DirectoryContents{
		Message: types.RespondTo(parent),
		Uids:    uids,
	}, nil)
	return nil
}

func (w *worker) emitMessageInfo(m *Message,
	parent types.WorkerMessage) error {
	info, err := m.MessageInfo()
	if err != nil {
		return fmt.Errorf("could not get MessageInfo: %v", err)
	}
	w.w.PostMessage(&types.MessageInfo{
		Message: types.RespondTo(parent),
		Info:    info,
	}, nil)
	return nil
}
