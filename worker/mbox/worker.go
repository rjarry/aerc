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

	"git.sr.ht/~rjarry/aerc/logging"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/handlers"
	"git.sr.ht/~rjarry/aerc/worker/lib"
	"git.sr.ht/~rjarry/aerc/worker/types"
	gomessage "github.com/emersion/go-message"
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
}

func NewWorker(worker *types.Worker) (types.Backend, error) {
	return &mboxWorker{
		worker: worker,
	}, nil
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
		w.data, err = createMailboxContainer(dir)
		if err != nil || w.data == nil {
			w.data = &mailboxContainer{
				mailboxes: make(map[string]*container),
			}
			reterr = err
			break
		} else {
			logging.Infof("configured with mbox file %s", dir)
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
					Name:       name,
					Attributes: nil,
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
		logging.Infof("%s opened", msg.Directory)

	case *types.FetchDirectoryContents:
		var infos []*models.MessageInfo
		for _, uid := range w.folder.Uids() {
			m, err := w.folder.Message(uid)
			if err != nil {
				logging.Errorf("could not get message %w", err)
				continue
			}
			info, err := lib.MessageInfo(m)
			if err != nil {
				logging.Errorf("could not get message info %w", err)
				continue
			}
			infos = append(infos, info)
		}
		uids, err := lib.Sort(infos, msg.SortCriteria)
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
		for _, uid := range msg.Uids {
			m, err := w.folder.Message(uid)
			if err != nil {
				reterr = err
				break
			}
			msgInfo, err := lib.MessageInfo(m)
			if err != nil {
				reterr = err
				break
			} else {
				w.worker.PostMessage(&types.MessageInfo{
					Message: types.RespondTo(msg),
					Info:    msgInfo,
				}, nil)
			}
		}
		w.worker.PostMessage(
			&types.Done{Message: types.RespondTo(msg)}, nil)

	case *types.FetchMessageBodyPart:
		m, err := w.folder.Message(msg.Uid)
		if err != nil {
			logging.Errorf("could not get message %d: %w", msg.Uid, err)
			reterr = err
			break
		}

		contentReader, err := m.NewReader()
		if err != nil {
			reterr = fmt.Errorf("could not get message reader: %w", err)
			break
		}

		fullMsg, err := gomessage.Read(contentReader)
		if err != nil {
			reterr = fmt.Errorf("could not read message: %w", err)
			break
		}

		r, err := lib.FetchEntityPartReader(fullMsg, msg.Part)
		if err != nil {
			logging.Errorf(
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
		for _, uid := range msg.Uids {
			m, err := w.folder.Message(uid)
			if err != nil {
				logging.Errorf("could not get message for uid %d: %w", uid, err)
				continue
			}
			r, err := m.NewReader()
			if err != nil {
				logging.Errorf("could not get message reader: %w", err)
				continue
			}
			defer r.Close()
			b, err := io.ReadAll(r)
			if err != nil {
				logging.Errorf("could not get message reader: %w", err)
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
		deleted := w.folder.Delete(msg.Uids)
		if len(deleted) > 0 {
			w.worker.PostMessage(&types.MessagesDeleted{
				Message: types.RespondTo(msg),
				Uids:    deleted,
			}, nil)
		}

		w.worker.PostMessage(&types.DirectoryInfo{
			Info: w.data.DirectoryInfo(w.name),
		}, nil)

		w.worker.PostMessage(
			&types.Done{Message: types.RespondTo(msg)}, nil)

	case *types.FlagMessages:
		for _, uid := range msg.Uids {
			m, err := w.folder.Message(uid)
			if err != nil {
				logging.Errorf("could not get message: %w", err)
				continue
			}
			if err := m.(*message).SetFlag(msg.Flag, msg.Enable); err != nil {
				logging.Errorf("could change flag %v to %t on message: %w", msg.Flag, msg.Enable, err)
				continue
			}
			info, err := lib.MessageInfo(m)
			if err != nil {
				logging.Errorf("could not get message info: %w", err)
				continue
			}

			w.worker.PostMessage(&types.MessageInfo{
				Message: types.RespondTo(msg),
				Info:    info,
			}, nil)
		}

		w.worker.PostMessage(&types.DirectoryInfo{
			Info: w.data.DirectoryInfo(w.name),
		}, nil)

		w.worker.PostMessage(
			&types.Done{Message: types.RespondTo(msg)}, nil)

	case *types.CopyMessages:
		err := w.data.Copy(msg.Destination, w.name, msg.Uids)
		if err != nil {
			reterr = err
			break
		}

		w.worker.PostMessage(&types.DirectoryInfo{
			Info: w.data.DirectoryInfo(w.name),
		}, nil)

		w.worker.PostMessage(&types.DirectoryInfo{
			Info: w.data.DirectoryInfo(msg.Destination),
		}, nil)

		w.worker.PostMessage(
			&types.Done{Message: types.RespondTo(msg)}, nil)

	case *types.SearchDirectory:
		criteria, err := lib.GetSearchCriteria(msg.Argv)
		if err != nil {
			reterr = err
			break
		}
		logging.Infof("Searching with parsed criteria: %#v", criteria)
		m := make([]lib.RawMessage, 0, len(w.folder.Uids()))
		for _, uid := range w.folder.Uids() {
			msg, err := w.folder.Message(uid)
			if err != nil {
				logging.Errorf("failed to get message for uid: %d", uid)
				continue
			}
			m = append(m, msg)
		}
		uids, err := lib.Search(m, criteria)
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
	for msg := range w.worker.Actions {
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
