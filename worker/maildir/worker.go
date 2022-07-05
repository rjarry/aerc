package maildir

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/emersion/go-maildir"
	"github.com/fsnotify/fsnotify"

	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/handlers"
	"git.sr.ht/~rjarry/aerc/worker/lib"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

func init() {
	handlers.RegisterWorkerFactory("maildir", NewWorker)
	handlers.RegisterWorkerFactory("maildirpp", NewMaildirppWorker)
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
	maildirpp           bool // whether to use Maildir++ directory layout
}

// NewWorker creates a new maildir worker with the provided worker.
func NewWorker(worker *types.Worker) (types.Backend, error) {
	watch, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("could not create file system watcher: %v", err)
	}
	return &Worker{worker: worker, watcher: watch}, nil
}

// NewMaildirppWorker creates a new Maildir++ worker with the provided worker.
func NewMaildirppWorker(worker *types.Worker) (types.Backend, error) {
	watch, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("could not create file system watcher: %v", err)
	}
	return &Worker{worker: worker, watcher: watch, maildirpp: true}, nil
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
	switch msg := msg.(type) {
	// Explicitly handle all asynchronous actions. Async actions are
	// responsible for posting their own Done message
	case *types.CheckMail:
		go w.handleCheckMail(msg)
	default:
		// Default handling, will be performed synchronously
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
}

func (w *Worker) handleFSEvent(ev fsnotify.Event) {
	// we only care about files being created, removed or renamed
	switch ev.Op {
	case fsnotify.Create, fsnotify.Remove, fsnotify.Rename:
		break
	default:
		return
	}
	// if there's not a selected directory to rescan, ignore
	if w.selected == nil {
		return
	}
	err := w.c.SyncNewMail(*w.selected)
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
	w.worker.PostMessage(&types.DirectoryInfo{
		Info: dirInfo,
	}, nil)
}

func (w *Worker) done(msg types.WorkerMessage) {
	w.worker.PostMessage(&types.Done{Message: types.RespondTo(msg)}, nil)
}

func (w *Worker) err(msg types.WorkerMessage, err error) {
	w.worker.PostMessage(&types.Error{
		Message: types.RespondTo(msg),
		Error:   err,
	}, nil)
}

func splitMaildirFile(name string) (uniq string, flags []maildir.Flag, err error) {
	i := strings.LastIndexByte(name, ':')
	if i < 0 {
		return "", nil, &maildir.MailfileError{Name: name}
	}
	info := name[i+1:]
	uniq = name[:i]
	if len(info) < 2 {
		return "", nil, &maildir.FlagError{Info: info, Experimental: false}
	}
	if info[1] != ',' || info[0] != '2' {
		return "", nil, &maildir.FlagError{Info: info, Experimental: false}
	}
	if info[0] == '1' {
		return "", nil, &maildir.FlagError{Info: info, Experimental: true}
	}
	flags = []maildir.Flag(info[2:])
	sort.Slice(flags, func(i, j int) bool { return info[i] < info[j] })
	return uniq, flags, nil
}

func dirFiles(name string) ([]string, error) {
	dir, err := os.Open(filepath.Join(name, "cur"))
	if err != nil {
		return nil, err
	}
	defer dir.Close()
	return dir.Readdirnames(-1)
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

		AccurateCounts: false,

		Caps: &models.Capabilities{
			Sort:   true,
			Thread: false,
		},
	}

	dir := w.c.Dir(name)
	var keyFlags map[string][]maildir.Flag
	files, err := dirFiles(string(dir))
	if err == nil {
		keyFlags = make(map[string][]maildir.Flag, len(files))
		for _, v := range files {
			key, flags, err := splitMaildirFile(v)
			if err != nil {
				w.worker.Logger.Printf("%q: error parsing flags (%q): %v", v, key, err)
				continue
			}
			keyFlags[key] = flags
		}
	} else {
		w.worker.Logger.Printf("disabled flags cache: %q: %v", dir, err)
	}

	uids, err := w.c.UIDs(dir)
	if err != nil {
		w.worker.Logger.Printf("could not get uids: %v", err)
		return dirInfo
	}

	dirInfo.Exists = len(uids)
	for _, uid := range uids {
		message, err := w.c.Message(dir, uid)
		if err != nil {
			w.worker.Logger.Printf("could not get message: %v", err)
			continue
		}
		var flags []maildir.Flag
		if keyFlags != nil {
			ok := false
			flags, ok = keyFlags[message.key]
			if !ok {
				w.worker.Logger.Printf("message (key=%q uid=%d) not found in map cache", message.key, message.uid)
				flags, err = message.Flags()
				if err != nil {
					w.worker.Logger.Printf("could not get flags: %v", err)
					continue
				}
			}
		} else {
			flags, err = message.Flags()
			if err != nil {
				w.worker.Logger.Printf("could not get flags: %v", err)
				continue
			}
		}
		seen := false
		for _, flag := range flags {
			if flag == maildir.FlagSeen {
				seen = true
				break
			}
		}
		if !seen {
			dirInfo.Unseen++
		}
		if w.c.IsRecent(uid) {
			dirInfo.Recent++
		}
	}
	dirInfo.AccurateCounts = true
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
	case *types.RemoveDirectory:
		return w.handleRemoveDirectory(msg)
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
	if len(dir) == 0 {
		return fmt.Errorf("could not resolve maildir from URL '%s'", msg.Config.Source)
	}
	c, err := NewContainer(dir, w.worker.Logger, w.maildirpp)
	if err != nil {
		w.worker.Logger.Printf("could not configure maildir: %s", dir)
		return err
	}
	w.c = c
	w.worker.Logger.Printf("configured base maildir: %s", dir)
	return nil
}

func (w *Worker) handleConnect(msg *types.Connect) error {
	return nil
}

func (w *Worker) handleListDirectories(msg *types.ListDirectories) error {
	// TODO If handleConfigure has returned error, w.c is nil.
	// It could be better if we skip directory listing completely
	// when configure fails.
	if w.c == nil {
		return errors.New("Incorrect maildir directory")
	}
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

	// remove existing watch paths
	if w.selected != nil {
		prevDir := filepath.Join(string(*w.selected), "new")
		if err := w.watcher.Remove(prevDir); err != nil {
			return fmt.Errorf("could not unwatch previous directory: %v", err)
		}
		prevDir = filepath.Join(string(*w.selected), "cur")
		if err := w.watcher.Remove(prevDir); err != nil {
			return fmt.Errorf("could not unwatch previous directory: %v", err)
		}
	}

	w.selected = &dir
	w.selectedName = msg.Directory

	// add watch paths
	newDir := filepath.Join(string(*w.selected), "new")
	if err := w.watcher.Add(newDir); err != nil {
		return fmt.Errorf("could not add watch to directory: %v", err)
	}
	newDir = filepath.Join(string(*w.selected), "cur")
	if err := w.watcher.Add(newDir); err != nil {
		return fmt.Errorf("could not add watch to directory: %v", err)
	}

	if err := dir.Clean(); err != nil {
		return fmt.Errorf("could not clean directory: %v", err)
	}

	info := &types.DirectoryInfo{
		Info: w.getDirectoryInfo(msg.Directory),
	}
	w.worker.PostMessage(info, nil)
	return nil
}

func (w *Worker) handleFetchDirectoryContents(
	msg *types.FetchDirectoryContents) error {
	var (
		uids []uint32
		err  error
	)
	if len(msg.FilterCriteria) > 0 {
		filter, err := parseSearch(msg.FilterCriteria)
		if err != nil {
			return err
		}
		uids, err = w.search(filter)
		if err != nil {
			return err
		}
	} else {
		uids, err = w.c.UIDs(*w.selected)
		if err != nil {
			w.worker.Logger.Printf("error scanning uids: %v", err)
			return err
		}
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
		info, err := w.msgInfoFromUid(uid)
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

func (w *Worker) handleRemoveDirectory(msg *types.RemoveDirectory) error {
	dir := w.c.Dir(msg.Directory)
	if err := os.RemoveAll(string(dir)); err != nil {
		w.worker.Logger.Printf("could not remove directory %s: %v",
			msg.Directory, err)
		return err
	}
	return nil
}

func (w *Worker) handleFetchMessageHeaders(
	msg *types.FetchMessageHeaders) error {
	for _, uid := range msg.Uids {
		info, err := w.msgInfoFromUid(uid)
		if err != nil {
			w.worker.Logger.Printf("could not get message info: %v", err)
			w.err(msg, err)
			continue
		}
		w.worker.PostMessage(&types.MessageInfo{
			Message: types.RespondTo(msg),
			Info:    info,
		}, nil)
		w.c.ClearRecentFlag(uid)
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
		defer r.Close()
		b, err := ioutil.ReadAll(r)
		if err != nil {
			return err
		}
		w.worker.PostMessage(&types.FullMessage{
			Message: types.RespondTo(msg),
			Content: &models.FullMessage{
				Uid:    uid,
				Reader: bytes.NewReader(b),
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
	}

	w.worker.PostMessage(&types.DirectoryInfo{
		Info: w.getDirectoryInfo(w.selectedName),
	}, nil)

	return nil
}

func (w *Worker) handleCopyMessages(msg *types.CopyMessages) error {
	dest := w.c.Dir(msg.Destination)
	err := w.c.CopyAll(dest, *w.selected, msg.Uids)
	if err != nil {
		return err
	}
	w.worker.PostMessage(&types.MessagesCopied{
		Message:     types.RespondTo(msg),
		Destination: msg.Destination,
		Uids:        msg.Uids,
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

func (w *Worker) msgInfoFromUid(uid uint32) (*models.MessageInfo, error) {
	m, err := w.c.Message(*w.selected, uid)
	if err != nil {
		return nil, err
	}
	info, err := m.MessageInfo()
	if err != nil {
		return nil, err
	}
	if w.c.IsRecent(uid) {
		info.Flags = append(info.Flags, models.RecentFlag)
	}
	return info, nil
}

func (w *Worker) handleCheckMail(msg *types.CheckMail) {
	ctx, cancel := context.WithTimeout(context.Background(), msg.Timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", "-c", msg.Command)
	ch := make(chan error)
	go func() {
		err := cmd.Run()
		ch <- err
	}()
	select {
	case <-ctx.Done():
		w.err(msg, fmt.Errorf("checkmail: timed out"))
	case err := <-ch:
		if err != nil {
			w.err(msg, fmt.Errorf("checkmail: error running command: %v", err))
		} else {
			w.done(msg)
		}
	}
}
