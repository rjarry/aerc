package mboxer

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"

	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/lib/rfc822"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/handlers"
	"git.sr.ht/~rjarry/aerc/worker/lib"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

func init() {
	handlers.RegisterWorkerFactory("mbox", NewWorker)
}

var errUnsupported = fmt.Errorf("unsupported command")

type mboxWorker struct {
	data   *mailboxContainer
	name   string
	folder *container
	worker *types.Worker

	capabilities   *models.Capabilities
	headers        []string
	headersExclude []string
}

func NewWorker(worker *types.Worker) (types.Backend, error) {
	return &mboxWorker{
		worker: worker,
		capabilities: &models.Capabilities{
			Sort:   true,
			Thread: false,
		},
	}, nil
}

// dir returns the container for the given directory name. If name is empty,
// returns the currently selected folder.
func (w *mboxWorker) dir(name string) *container {
	if name != "" {
		if f, ok := w.data.Mailbox(name); ok {
			return f
		}
	}
	return w.folder
}

// dirName returns the directory name. If name is empty, returns the currently
// selected directory name.
func (w *mboxWorker) dirName(name string) string {
	if name != "" {
		return name
	}
	return w.name
}

func (w *mboxWorker) handleMessage(msg types.WorkerMessage) error {
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
		if u.Host == "" && u.Path == "" {
			u, err = url.Parse("mbox://" + u.Opaque)
			if err != nil {
				reterr = err
				break
			}
		}

		var dir string
		if u.Host == "~" {
			home, err := os.UserHomeDir()
			if err != nil {
				reterr = err
				break
			}
			dir = filepath.Join(home, u.Path)
		} else {
			dir = filepath.Join(u.Host, u.Path)
		}
		w.headers = msg.Config.Headers
		w.headersExclude = msg.Config.HeadersExclude
		w.data, err = createMailboxContainer(dir)
		if err != nil || w.data == nil {
			w.data = &mailboxContainer{
				mailboxes: make(map[string]*container),
			}
			reterr = err
			break
		} else {
			w.worker.Debugf("configured with mbox file %s", dir)
		}

	case *types.Connect, *types.Reconnect, *types.Disconnect:
		w.worker.PostMessage(&types.Done{Message: types.RespondTo(msg)}, nil)

	case *types.ListDirectories:
		dirs := w.data.Names()
		sort.Strings(dirs)
		for _, name := range dirs {
			w.worker.PostMessage(&types.Directory{
				Message: types.RespondTo(msg),
				Dir: &models.Directory{
					Name: name,
				},
			}, nil)
			w.worker.PostMessage(&types.DirectoryInfo{
				Info: w.data.DirectoryInfo(name),
			}, nil)
		}
		w.worker.PostMessage(&types.Done{Message: types.RespondTo(msg)}, nil)

	case *types.OpenDirectory:
		w.name = msg.Directory
		var ok bool
		w.folder, ok = w.data.Mailbox(w.name)
		if !ok {
			w.folder = w.data.Create(w.name)
			w.worker.PostMessage(&types.Done{
				Message: types.RespondTo(&types.CreateDirectory{}),
			}, nil)
		}
		w.worker.PostMessage(&types.DirectoryInfo{
			Info: w.data.DirectoryInfo(msg.Directory),
		}, nil)
		w.worker.PostMessage(&types.Done{Message: types.RespondTo(msg)}, nil)
		w.worker.Debugf("%s opened", msg.Directory)

	case *types.FetchDirectoryContents:
		folder := w.dir(msg.Directory)
		uids, err := filterUids(folder, folder.Uids(), msg.Filter)
		if err != nil {
			reterr = err
			break
		}
		uids, err = sortUids(folder, uids, msg.SortCriteria)
		if err != nil {
			reterr = err
			break
		}
		if len(uids) == 0 {
			reterr = fmt.Errorf("mbox: no uids in directory")
			break
		}
		w.worker.PostMessage(&types.DirectoryContents{
			Message: types.RespondTo(msg),
			Uids:    uids,
		}, nil)
		w.worker.PostMessage(&types.Done{Message: types.RespondTo(msg)}, nil)

	case *types.FetchDirectoryThreaded:
		reterr = errUnsupported

	case *types.CreateDirectory:
		w.data.Create(msg.Directory)
		w.worker.PostMessage(&types.Done{Message: types.RespondTo(msg)}, nil)

	case *types.RemoveDirectory:
		if err := w.data.Remove(msg.Directory); err != nil {
			reterr = err
			break
		}
		w.worker.PostMessage(&types.Done{Message: types.RespondTo(msg)}, nil)

	case *types.FetchMessageHeaders:
		folder := w.dir(msg.Directory)
		for _, uid := range msg.Uids {
			m, err := folder.Message(uid)
			if err != nil {
				reterr = err
				break
			}
			msgInfo, err := messageInfo(m, true)
			if err != nil {
				w.worker.PostMessage(&types.MessageInfo{
					Info: &models.MessageInfo{
						Envelope: &models.Envelope{},
						Flags:    models.SeenFlag,
						Uid:      uid,
						Error:    err,
					},
					Message: types.RespondTo(msg),
				}, nil)
				continue
			} else {
				switch {
				case len(w.headersExclude) > 0:
					msgInfo.RFC822Headers = lib.LimitHeaders(msgInfo.RFC822Headers, w.headersExclude, true)
				case len(w.headers) > 0:
					msgInfo.RFC822Headers = lib.LimitHeaders(msgInfo.RFC822Headers, w.headers, false)
				}
				w.worker.PostMessage(&types.MessageInfo{
					Message: types.RespondTo(msg),
					Info:    msgInfo,
				}, nil)
			}
		}
		w.worker.PostMessage(
			&types.Done{Message: types.RespondTo(msg)}, nil)

	case *types.FetchMessageBodyPart:
		folder := w.dir(msg.Directory)
		m, err := folder.Message(msg.Uid)
		if err != nil {
			w.worker.Errorf("could not get message %d: %v", msg.Uid, err)
			reterr = err
			break
		}

		contentReader, err := m.NewReader()
		if err != nil {
			reterr = fmt.Errorf("could not get message reader: %w", err)
			break
		}

		fullMsg, err := rfc822.ReadMessage(contentReader)
		if err != nil {
			reterr = fmt.Errorf("could not read message: %w", err)
			break
		}

		r, err := rfc822.FetchEntityPartReader(fullMsg, msg.Part)
		if err != nil {
			w.worker.Errorf(
				"could not get body part reader for message=%d, parts=%#v: %w",
				msg.Uid, msg.Part, err)
			reterr = err
			break
		}

		w.worker.PostMessage(&types.MessageBodyPart{
			Message: types.RespondTo(msg),
			Part: &models.MessageBodyPart{
				Reader: r,
				Uid:    msg.Uid,
			},
		}, nil)

	case *types.FetchFullMessages:
		folder := w.dir(msg.Directory)
		for _, uid := range msg.Uids {
			m, err := folder.Message(uid)
			if err != nil {
				w.worker.Errorf("could not get message for uid %d: %v", uid, err)
				continue
			}
			r, err := m.NewReader()
			if err != nil {
				w.worker.Errorf("could not get message reader: %v", err)
				continue
			}
			defer r.Close()
			b, err := io.ReadAll(r)
			if err != nil {
				w.worker.Errorf("could not get message reader: %v", err)
				continue
			}
			w.worker.PostMessage(&types.FullMessage{
				Message: types.RespondTo(msg),
				Content: &models.FullMessage{
					Uid:    uid,
					Reader: bytes.NewReader(b),
				},
			}, nil)
		}
		w.worker.PostMessage(&types.Done{
			Message: types.RespondTo(msg),
		}, nil)

	case *types.DeleteMessages:
		folder := w.dir(msg.Directory)
		deleted := folder.Delete(msg.Uids)
		if len(deleted) > 0 {
			w.worker.PostMessage(&types.MessagesDeleted{
				Message: types.RespondTo(msg),
				Uids:    deleted,
			}, nil)
		}

		w.worker.PostMessage(&types.DirectoryInfo{
			Info: w.data.DirectoryInfo(w.dirName(msg.Directory)),
		}, nil)

		w.worker.PostMessage(
			&types.Done{Message: types.RespondTo(msg)}, nil)

	case *types.FlagMessages:
		folder := w.dir(msg.Directory)
		for _, uid := range msg.Uids {
			m, err := folder.Message(uid)
			if err != nil {
				w.worker.Errorf("could not get message: %v", err)
				continue
			}
			if err := m.(*message).SetFlag(msg.Flags, msg.Enable); err != nil {
				w.worker.Errorf("could not change flag %v to %t on message: %v",
					msg.Flags, msg.Enable, err)
				continue
			}
			info, err := rfc822.MessageInfo(m)
			if err != nil {
				w.worker.Errorf("could not get message info: %v", err)
				continue
			}

			w.worker.PostMessage(&types.MessageInfo{
				Message: types.RespondTo(msg),
				Info:    info,
			}, nil)
		}

		w.worker.PostMessage(&types.DirectoryInfo{
			Info: w.data.DirectoryInfo(w.dirName(msg.Directory)),
		}, nil)

		w.worker.PostMessage(
			&types.Done{Message: types.RespondTo(msg)}, nil)

	case *types.CopyMessages:
		src := w.dirName(msg.Source)
		err := w.data.Copy(msg.Destination, src, msg.Uids)
		if err != nil {
			reterr = err
			break
		}

		w.worker.PostMessage(&types.DirectoryInfo{
			Info: w.data.DirectoryInfo(src),
		}, nil)

		w.worker.PostMessage(&types.DirectoryInfo{
			Info: w.data.DirectoryInfo(msg.Destination),
		}, nil)

		w.worker.PostMessage(
			&types.Done{Message: types.RespondTo(msg)}, nil)
	case *types.MoveMessages:
		src := w.dirName(msg.Source)
		srcFolder := w.dir(msg.Source)
		err := w.data.Copy(msg.Destination, src, msg.Uids)
		if err != nil {
			reterr = err
			break
		}
		deleted := srcFolder.Delete(msg.Uids)
		if len(deleted) > 0 {
			w.worker.PostMessage(&types.MessagesDeleted{
				Message: types.RespondTo(msg),
				Uids:    deleted,
			}, nil)
		}
		w.worker.PostMessage(&types.DirectoryInfo{
			Info: w.data.DirectoryInfo(msg.Destination),
		}, nil)
		w.worker.PostMessage(
			&types.Done{Message: types.RespondTo(msg)}, nil)

	case *types.SearchDirectory:
		folder := w.dir(msg.Directory)
		uids, err := filterUids(folder, folder.Uids(), msg.Criteria)
		if err != nil {
			reterr = err
			break
		}
		w.worker.PostMessage(&types.SearchResults{
			Message: types.RespondTo(msg),
			Uids:    uids,
		}, nil)

	case *types.AppendMessage:
		if msg.Destination == "" {
			reterr = fmt.Errorf("AppendMessage with empty destination directory")
			break
		}
		folder, ok := w.data.Mailbox(msg.Destination)
		if !ok {
			folder = w.data.Create(msg.Destination)
			w.worker.PostMessage(&types.Done{
				Message: types.RespondTo(&types.CreateDirectory{}),
			}, nil)
		}

		if err := folder.Append(msg.Reader, msg.Flags); err != nil {
			reterr = err
			break
		} else {
			w.worker.PostMessage(&types.DirectoryInfo{
				Info: w.data.DirectoryInfo(msg.Destination),
			}, nil)
			w.worker.PostMessage(&types.Done{Message: types.RespondTo(msg)}, nil)
		}

	case *types.AnsweredMessages:
		reterr = errUnsupported
	default:
		reterr = errUnsupported
	}

	return reterr
}

func (w *mboxWorker) Run() {
	defer log.PanicHandler()
	for msg := range w.worker.Actions() {
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
	}
}

func (w *mboxWorker) Capabilities() *models.Capabilities {
	return w.capabilities
}

func (w *mboxWorker) PathSeparator() string {
	return "/"
}

func filterUids(folder *container, uids []models.UID, criteria *types.SearchCriteria) ([]models.UID, error) {
	log.Debugf("Search with parsed criteria: %#v", criteria)
	m := make([]rfc822.RawMessage, 0, len(uids))
	for _, uid := range uids {
		msg, err := folder.Message(uid)
		if err != nil {
			log.Errorf("failed to get message for uid: %d", uid)
			continue
		}
		m = append(m, msg)
	}
	return lib.Search(m, criteria)
}

func sortUids(folder *container, uids []models.UID,
	criteria []*types.SortCriterion,
) ([]models.UID, error) {
	var infos []*models.MessageInfo
	needSize := false
	for _, item := range criteria {
		if item.Field == types.SortSize {
			needSize = true
		}
	}
	for _, uid := range uids {
		m, err := folder.Message(uid)
		if err != nil {
			log.Errorf("could not get message %v", err)
			continue
		}
		info, err := messageInfo(m, needSize)
		if err != nil {
			log.Errorf("could not get message info %v", err)
			continue
		}
		infos = append(infos, info)
	}
	return lib.Sort(infos, criteria)
}

func messageInfo(m rfc822.RawMessage, needSize bool) (*models.MessageInfo, error) {
	info, err := rfc822.MessageInfo(m)
	if err != nil {
		return nil, err
	}
	if !needSize {
		return info, nil
	}
	r, err := m.NewReader()
	if err != nil {
		return nil, err
	}
	size, err := io.Copy(io.Discard, r)
	if err != nil {
		return nil, err
	}
	info.Size = uint32(size)
	return info, nil
}
