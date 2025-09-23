package account

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/lib/xdg"
	"git.sr.ht/~rjarry/aerc/models"
	mboxer "git.sr.ht/~rjarry/aerc/worker/mbox"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

type ExportMbox struct {
	Filename string `opt:"filename" complete:"CompleteFilename" desc:"Output file path."`
}

func init() {
	commands.Register(ExportMbox{})
}

func (ExportMbox) Description() string {
	return "Export messages in the current folder to an mbox file."
}

func (ExportMbox) Context() commands.CommandContext {
	return commands.MESSAGE_LIST
}

func (ExportMbox) Aliases() []string {
	return []string{"export-mbox"}
}

func (*ExportMbox) CompleteFilename(arg string) []string {
	return commands.CompletePath(arg, false)
}

func (e ExportMbox) Execute(args []string) error {
	acct := app.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	store := acct.Store()
	if store == nil {
		return errors.New("No message store selected")
	}

	e.Filename = xdg.ExpandHome(e.Filename)

	fi, err := os.Stat(e.Filename)
	if err == nil && fi.IsDir() {
		if path := acct.SelectedDirectory(); path != "" {
			if f := filepath.Base(path); f != "" {
				e.Filename = filepath.Join(e.Filename, f+".mbox")
			}
		}
	}

	app.PushStatus("Exporting to "+e.Filename, 10*time.Second)

	// uids of messages to export
	var uids []models.UID

	// check if something is marked - we export that then
	msgProvider, ok := app.SelectedTabContent().(app.ProvidesMessages)
	if !ok {
		msgProvider = app.SelectedAccount()
	}
	if msgProvider != nil {
		marked, err := msgProvider.MarkedMessages()
		if err == nil && len(marked) > 0 {
			uids, err = sortMarkedUids(marked, store)
			if err != nil {
				return err
			}
		}
	}

	// if no messages were marked, we export everything
	if len(uids) == 0 {
		var err error
		uids, err = sortAllUids(store)
		if err != nil {
			return err
		}
	}

	go func() {
		defer log.PanicHandler()
		file, err := os.Create(e.Filename)
		if err != nil {
			log.Errorf("failed to create file: %v", err)
			app.PushError(err.Error())
			return
		}
		defer file.Close()

		var mu sync.Mutex
		var ctr uint
		var retries int

		done := make(chan bool)

		t := time.Now()
		total := len(uids)

		for len(uids) > 0 {
			if retries > 0 {
				if retries > 10 {
					errorMsg := fmt.Sprintf("too many retries: %d; stopping export", retries)
					log.Errorf(errorMsg)
					app.PushError(args[0] + " " + errorMsg)
					break
				}
				sleeping := time.Duration(retries * 1e9 * 2)
				log.Debugf("sleeping for %s before retrying; retries: %d", sleeping, retries)
				time.Sleep(sleeping)
			}

			// do not modify uids directly as it is used inside the worker action
			pendingUids := make([]models.UID, len(uids))
			copy(pendingUids, uids)

			log.Debugf("fetching %d for export", len(uids))
			acct.Worker().PostAction(&types.FetchFullMessages{
				Uids: uids,
			}, func(msg types.WorkerMessage) {
				switch msg := msg.(type) {
				case *types.Done:
					done <- true
				case *types.Error:
					log.Errorf("failed to fetch message: %v", msg.Error)
					app.PushError(args[0] + " error encountered: " + msg.Error.Error())
					done <- false
				case *types.FullMessage:
					mu.Lock()
					err := mboxer.Write(file, msg.Content.Reader, "", t)
					if err != nil {
						log.Warnf("failed to write mbox: %v", err)
					}
					for i, uid := range pendingUids {
						if uid == msg.Content.Uid {
							pendingUids = append(pendingUids[:i], pendingUids[i+1:]...)
							break
						}
					}
					ctr++
					mu.Unlock()
				}
			})
			if ok := <-done; ok {
				break
			}
			uids = pendingUids
			retries++
		}
		statusInfo := fmt.Sprintf("Exported %d of %d messages to %s.", ctr, total, e.Filename)
		app.PushStatus(statusInfo, 10*time.Second)
		log.Debugf(statusInfo)
	}()

	return nil
}

func sortMarkedUids(marked []models.UID, store *lib.MessageStore) ([]models.UID, error) {
	lookup := map[models.UID]bool{}
	for _, uid := range marked {
		lookup[uid] = true
	}
	uids := []models.UID{}
	iter := store.UidsIterator()
	for iter.Next() {
		uid, ok := iter.Value().(models.UID)
		if !ok {
			return nil, errors.New("Invalid message UID value")
		}
		_, marked := lookup[uid]
		if marked {
			uids = append(uids, uid)
		}
	}
	return uids, nil
}

func sortAllUids(store *lib.MessageStore) ([]models.UID, error) {
	uids := []models.UID{}
	iter := store.UidsIterator()
	for iter.Next() {
		uid, ok := iter.Value().(models.UID)
		if !ok {
			return nil, errors.New("Invalid message UID value")
		}
		uids = append(uids, uid)
	}
	return uids, nil
}
