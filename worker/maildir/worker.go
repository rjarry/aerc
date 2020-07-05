package maildir

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"

	"github.com/emersion/go-maildir"
	"github.com/fsnotify/fsnotify"

	"git.sr.ht/~sircmpwn/aerc/models"
	"git.sr.ht/~sircmpwn/aerc/worker/handlers"
	"git.sr.ht/~sircmpwn/aerc/worker/lib"
	"git.sr.ht/~sircmpwn/aerc/worker/types"
)

func init() {
	handlers.RegisterWorkerFactory("maildir", NewWorker)
}

var errUnsupported = fmt.Errorf("unsupported command")

// A Worker handles interfacing between aerc's UI and a group of maildirs.
type Worker struct {
	c                   *Container
	selected            *maildir.Dir
	selectedName        string
	worker              *types.Worker
	watcher             *fsnotify.Watcher
	currentSortCriteria []*types.SortCriterion
}

// NewWorker creates a new maildir worker with the provided worker.
func NewWorker(worker *types.Worker) (types.Backend, error) {
	watch, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("could not create file system watcher: %v", err)
	}
	return &Worker{worker: worker, watcher: watch}, nil
}

// Run starts the worker's message handling loop.
func (w *Worker) Run() {
	for {
		select {
		case action := <-w.worker.Actions:
			w.handleAction(action)
		case ev := <-w.watcher.Events:
			w.handleFSEvent(ev)
		}
	}
}

func (w *Worker) handleAction(action types.WorkerMessage) {
	msg := w.worker.ProcessAction(action)
	if err := w.handleMessage(msg); err == errUnsupported {
		w.worker.PostMessage(&types.Unsupported{
			Message: types.RespondTo(msg),
		}, nil)
	} else if err != nil {
		w.worker.PostMessage(&types.Error{
			Message: types.RespondTo(msg),
			Error:   err,
		}, nil)
	} else {
		w.done(msg)
	}
}

func (w *Worker) handleFSEvent(ev fsnotify.Event) {
	// we only care about files being created
	if ev.Op != fsnotify.Create {
		return
	}
	// if there's not a selected directory to rescan, ignore
	if w.selected == nil {
		return
	}
	newUnseen, err := w.selected.Unseen()
	if err != nil {
		w.worker.Logger.Printf("could not move new to cur : %v", err)
		return
	}
	uids, err := w.c.UIDs(*w.selected)
	if err != nil {
		w.worker.Logger.Printf("could not scan UIDs: %v", err)
		return
	}
	sortedUids, err := w.sort(uids, w.currentSortCriteria)
	if err != nil {
		w.worker.Logger.Printf("error sorting directory: %v", err)
		return
	}
	w.worker.PostMessage(&types.DirectoryContents{
		Uids: sortedUids,
	}, nil)
	dirInfo := w.getDirectoryInfo(w.selectedName)
	dirInfo.Recent = len(newUnseen)
	w.worker.PostMessage(&types.DirectoryInfo{
		Info: dirInfo,
	}, nil)
}

func (w *Worker) done(msg types.WorkerMessage) {
	w.worker.PostMessage(&types.Done{types.RespondTo(msg)}, nil)
}

func (w *Worker) err(msg types.WorkerMessage, err error) {
	w.worker.PostMessage(&types.Error{
		Message: types.RespondTo(msg),
		Error:   err,
	}, nil)
}

func (w *Worker) getDirectoryInfo(name string) *models.DirectoryInfo {
	dirInfo := &models.DirectoryInfo{
		Name:     name,
		Flags:    []string{},
		ReadOnly: false,
		// total messages
		Exists: 0,
		// new messages since mailbox was last opened
		Recent: 0,
		// total unread
		Unseen: 0,

		AccurateCounts: true,
	}

	dir := w.c.Dir(name)

	uids, err := w.c.UIDs(dir)
	if err != nil {
		w.worker.Logger.Printf("could not get uids: %v", err)
		return dirInfo
	}

	recent, err := dir.UnseenCount()
	if err != nil {
		w.worker.Logger.Printf("could not get unseen count: %v", err)
	}
	dirInfo.Recent = recent

	for _, uid := range uids {
		message, err := w.c.Message(dir, uid)
		if err != nil {
			w.worker.Logger.Printf("could not get message: %v", err)
			continue
		}
		flags, err := message.Flags()
		if err != nil {
			w.worker.Logger.Printf("could not get flags: %v", err)
			continue
		}
		seen := false
		for _, flag := range flags {
			if flag == maildir.FlagSeen {
				seen = true
			}
		}
		if !seen {
			dirInfo.Unseen++
		}
	}
	dirInfo.Unseen += dirInfo.Recent
	dirInfo.Exists = len(uids) + recent
	return dirInfo
}

func (w *Worker) handleMessage(msg types.WorkerMessage) error {
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
	case *types.CreateDirectory:
		return w.handleCreateDirectory(msg)
	case *types.FetchMessageHeaders:
		return w.handleFetchMessageHeaders(msg)
	case *types.FetchMessageBodyPart:
		return w.handleFetchMessageBodyPart(msg)
	case *types.FetchFullMessages:
		return w.handleFetchFullMessages(msg)
	case *types.DeleteMessages:
		return w.handleDeleteMessages(msg)
	case *types.FlagMessages:
		return w.handleFlagMessages(msg)
	case *types.AnsweredMessages:
		return w.handleAnsweredMessages(msg)
	case *types.CopyMessages:
		return w.handleCopyMessages(msg)
	case *types.AppendMessage:
		return w.handleAppendMessage(msg)
	case *types.SearchDirectory:
		return w.handleSearchDirectory(msg)
	}
	return errUnsupported
}

func (w *Worker) handleConfigure(msg *types.Configure) error {
	u, err := url.Parse(msg.Config.Source)
	if err != nil {
		w.worker.Logger.Printf("error configuring maildir worker: %v", err)
		return err
	}
	dir := u.Path
	if u.Host == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("could not resolve home directory: %v", err)
		}
		dir = filepath.Join(home, u.Path)
	}
	w.c = NewContainer(dir, w.worker.Logger)
	w.worker.Logger.Printf("configured base maildir: %s", dir)
	return nil
}

func (w *Worker) handleConnect(msg *types.Connect) error {
	return nil
}

func (w *Worker) handleListDirectories(msg *types.ListDirectories) error {
	dirs, err := w.c.ListFolders()
	if err != nil {
		w.worker.Logger.Printf("error listing directories: %v", err)
		return err
	}
	for _, name := range dirs {
		w.worker.PostMessage(&types.Directory{
			Message: types.RespondTo(msg),
			Dir: &models.Directory{
				Name:       name,
				Attributes: []string{},
			},
		}, nil)

		w.worker.PostMessage(&types.DirectoryInfo{
			Info: w.getDirectoryInfo(name),
		}, nil)
	}
	return nil
}

func (w *Worker) handleOpenDirectory(msg *types.OpenDirectory) error {
	w.worker.Logger.Printf("opening %s", msg.Directory)

	// open the directory
	dir, err := w.c.OpenDirectory(msg.Directory)
	if err != nil {
		return err
	}

	// remove existing watch path
	if w.selected != nil {
		prevDir := filepath.Join(string(*w.selected), "new")
		if err := w.watcher.Remove(prevDir); err != nil {
			return fmt.Errorf("could not unwatch previous directory: %v", err)
		}
	}

	w.selected = &dir
	w.selectedName = msg.Directory

	// add watch path
	newDir := filepath.Join(string(*w.selected), "new")
	if err := w.watcher.Add(newDir); err != nil {
		return fmt.Errorf("could not add watch to directory: %v", err)
	}

	if err := dir.Clean(); err != nil {
		return fmt.Errorf("could not clean directory: %v", err)
	}

	// TODO: why does this need to be sent twice??
	info := &types.DirectoryInfo{
		Info: w.getDirectoryInfo(msg.Directory),
	}
	w.worker.PostMessage(info, nil)
	w.worker.PostMessage(info, nil)
	return nil
}

func (w *Worker) handleFetchDirectoryContents(
	msg *types.FetchDirectoryContents) error {
	uids, err := w.c.UIDs(*w.selected)
	if err != nil {
		w.worker.Logger.Printf("error scanning uids: %v", err)
		return err
	}
	sortedUids, err := w.sort(uids, msg.SortCriteria)
	if err != nil {
		w.worker.Logger.Printf("error sorting directory: %v", err)
		return err
	}
	w.currentSortCriteria = msg.SortCriteria
	w.worker.PostMessage(&types.DirectoryContents{
		Message: types.RespondTo(msg),
		Uids:    sortedUids,
	}, nil)
	return nil
}

func (w *Worker) sort(uids []uint32, criteria []*types.SortCriterion) ([]uint32, error) {
	if len(criteria) == 0 {
		return uids, nil
	}
	var msgInfos []*models.MessageInfo
	for _, uid := range uids {
		m, err := w.c.Message(*w.selected, uid)
		if err != nil {
			w.worker.Logger.Printf("could not get message: %v", err)
			continue
		}
		info, err := m.MessageInfo()
		if err != nil {
			w.worker.Logger.Printf("could not get message info: %v", err)
			continue
		}
		msgInfos = append(msgInfos, info)
	}
	sortedUids, err := lib.Sort(msgInfos, criteria)
	if err != nil {
		w.worker.Logger.Printf("could not sort the messages: %v", err)
		return nil, err
	}
	return sortedUids, nil
}

func (w *Worker) handleCreateDirectory(msg *types.CreateDirectory) error {
	dir := w.c.Dir(msg.Directory)
	if err := dir.Init(); err != nil {
		w.worker.Logger.Printf("could not create directory %s: %v",
			msg.Directory, err)
		return err
	}
	return nil
}

func (w *Worker) handleFetchMessageHeaders(
	msg *types.FetchMessageHeaders) error {
	for _, uid := range msg.Uids {
		m, err := w.c.Message(*w.selected, uid)
		if err != nil {
			w.worker.Logger.Printf("could not get message: %v", err)
			w.err(msg, err)
			continue
		}
		info, err := m.MessageInfo()
		if err != nil {
			w.worker.Logger.Printf("could not get message info: %v", err)
			w.err(msg, err)
			continue
		}
		w.worker.PostMessage(&types.MessageInfo{
			Message: types.RespondTo(msg),
			Info:    info,
		}, nil)
	}
	return nil
}

func (w *Worker) handleFetchMessageBodyPart(
	msg *types.FetchMessageBodyPart) error {

	// get reader
	m, err := w.c.Message(*w.selected, msg.Uid)
	if err != nil {
		w.worker.Logger.Printf("could not get message %d: %v", msg.Uid, err)
		return err
	}
	r, err := m.NewBodyPartReader(msg.Part)
	if err != nil {
		w.worker.Logger.Printf(
			"could not get body part reader for message=%d, parts=%#v: %v",
			msg.Uid, msg.Part, err)
		return err
	}
	w.worker.PostMessage(&types.MessageBodyPart{
		Message: types.RespondTo(msg),
		Part: &models.MessageBodyPart{
			Reader: r,
			Uid:    msg.Uid,
		},
	}, nil)

	return nil
}

func (w *Worker) handleFetchFullMessages(msg *types.FetchFullMessages) error {
	for _, uid := range msg.Uids {
		m, err := w.c.Message(*w.selected, uid)
		if err != nil {
			w.worker.Logger.Printf("could not get message %d: %v", uid, err)
			return err
		}
		r, err := m.NewReader()
		if err != nil {
			w.worker.Logger.Printf("could not get message reader: %v", err)
			return err
		}
		w.worker.PostMessage(&types.FullMessage{
			Message: types.RespondTo(msg),
			Content: &models.FullMessage{
				Uid:    uid,
				Reader: r,
			},
		}, nil)
	}
	return nil
}

func (w *Worker) handleDeleteMessages(msg *types.DeleteMessages) error {
	deleted, err := w.c.DeleteAll(*w.selected, msg.Uids)
	if len(deleted) > 0 {
		w.worker.PostMessage(&types.MessagesDeleted{
			Message: types.RespondTo(msg),
			Uids:    deleted,
		}, nil)
	}
	if err != nil {
		w.worker.Logger.Printf("error removing some messages: %v", err)
		return err
	}

	w.worker.PostMessage(&types.DirectoryInfo{
		Info: w.getDirectoryInfo(w.selectedName),
	}, nil)

	return nil
}

func (w *Worker) handleAnsweredMessages(msg *types.AnsweredMessages) error {
	for _, uid := range msg.Uids {
		m, err := w.c.Message(*w.selected, uid)
		if err != nil {
			w.worker.Logger.Printf("could not get message: %v", err)
			w.err(msg, err)
			continue
		}
		if err := m.MarkReplied(msg.Answered); err != nil {
			w.worker.Logger.Printf(
				"could not mark message as answered: %v", err)
			w.err(msg, err)
			continue
		}
		info, err := m.MessageInfo()
		if err != nil {
			w.worker.Logger.Printf("could not get message info: %v", err)
			w.err(msg, err)
			continue
		}

		w.worker.PostMessage(&types.MessageInfo{
			Message: types.RespondTo(msg),
			Info:    info,
		}, nil)

		w.worker.PostMessage(&types.DirectoryInfo{
			Info: w.getDirectoryInfo(w.selectedName),
		}, nil)
	}
	return nil
}

func (w *Worker) handleFlagMessages(msg *types.FlagMessages) error {
	for _, uid := range msg.Uids {
		m, err := w.c.Message(*w.selected, uid)
		if err != nil {
			w.worker.Logger.Printf("could not get message: %v", err)
			w.err(msg, err)
			continue
		}
		flag := flagToMaildir[msg.Flag]
		if err := m.SetOneFlag(flag, msg.Enable); err != nil {
			w.worker.Logger.Printf("could change flag %v to %v on message: %v", flag, msg.Enable, err)
			w.err(msg, err)
			continue
		}
		info, err := m.MessageInfo()
		if err != nil {
			w.worker.Logger.Printf("could not get message info: %v", err)
			w.err(msg, err)
			continue
		}

		w.worker.PostMessage(&types.MessageInfo{
			Message: types.RespondTo(msg),
			Info:    info,
		}, nil)

		w.worker.PostMessage(&types.DirectoryInfo{
			Info: w.getDirectoryInfo(w.selectedName),
		}, nil)
	}
	return nil
}

func (w *Worker) handleCopyMessages(msg *types.CopyMessages) error {
	dest := w.c.Dir(msg.Destination)
	err := w.c.CopyAll(dest, *w.selected, msg.Uids)
	if err != nil {
		return err
	}

	w.worker.PostMessage(&types.DirectoryInfo{
		Info: w.getDirectoryInfo(w.selectedName),
	}, nil)

	w.worker.PostMessage(&types.DirectoryInfo{
		Info: w.getDirectoryInfo(msg.Destination),
	}, nil)

	return nil
}

func (w *Worker) handleAppendMessage(msg *types.AppendMessage) error {
	// since we are the "master" maildir process, we can modify the maildir directly
	dest := w.c.Dir(msg.Destination)
	_, writer, err := dest.Create(translateFlags(msg.Flags))
	if err != nil {
		w.worker.Logger.Printf("could not create message at %s: %v",
			msg.Destination, err)
		return err
	}
	defer writer.Close()
	if _, err := io.Copy(writer, msg.Reader); err != nil {
		w.worker.Logger.Printf("could not write message to destination: %v", err)
		return err
	}

	w.worker.PostMessage(&types.DirectoryInfo{
		Info: w.getDirectoryInfo(msg.Destination),
	}, nil)
	return nil
}

func (w *Worker) handleSearchDirectory(msg *types.SearchDirectory) error {
	w.worker.Logger.Printf("Searching directory %v with args: %v", *w.selected, msg.Argv)
	criteria, err := parseSearch(msg.Argv)
	if err != nil {
		return err
	}
	w.worker.Logger.Printf("Searching with parsed criteria: %#v", criteria)
	uids, err := w.search(criteria)
	if err != nil {
		return err
	}
	w.worker.PostMessage(&types.SearchResults{
		Message: types.RespondTo(msg),
		Uids:    uids,
	}, nil)
	return nil
}
