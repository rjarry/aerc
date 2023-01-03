package msg

import (
	"errors"
	"fmt"
	"path"
	"sync"

	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/log"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/widgets"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

const (
	ARCHIVE_FLAT  = "flat"
	ARCHIVE_YEAR  = "year"
	ARCHIVE_MONTH = "month"
)

type Archive struct{}

func init() {
	register(Archive{})
}

func (Archive) Aliases() []string {
	return []string{"archive"}
}

func (Archive) Complete(aerc *widgets.Aerc, args []string) []string {
	valid := []string{"flat", "year", "month"}
	return commands.CompletionFromList(aerc, valid, args)
}

func (Archive) Execute(aerc *widgets.Aerc, args []string) error {
	if len(args) != 2 {
		return errors.New("Usage: archive <flat|year|month>")
	}
	h := newHelper(aerc)
	msgs, err := h.messages()
	if err != nil {
		return err
	}
	err = archive(aerc, msgs, args[1])
	return err
}

func archive(aerc *widgets.Aerc, msgs []*models.MessageInfo, archiveType string) error {
	h := newHelper(aerc)
	acct, err := h.account()
	if err != nil {
		return err
	}
	store, err := h.store()
	if err != nil {
		return err
	}
	var uids []uint32
	for _, msg := range msgs {
		uids = append(uids, msg.Uid)
	}
	archiveDir := acct.AccountConfig().Archive
	marker := store.Marker()
	marker.ClearVisualMark()
	next := findNextNonDeleted(uids, store)

	var uidMap map[string][]uint32
	switch archiveType {
	case ARCHIVE_MONTH:
		uidMap = groupBy(msgs, func(msg *models.MessageInfo) string {
			dir := path.Join(archiveDir,
				fmt.Sprintf("%d", msg.Envelope.Date.Year()),
				fmt.Sprintf("%02d", msg.Envelope.Date.Month()))
			return dir
		})
	case ARCHIVE_YEAR:
		uidMap = groupBy(msgs, func(msg *models.MessageInfo) string {
			dir := path.Join(archiveDir, fmt.Sprintf("%v",
				msg.Envelope.Date.Year()))
			return dir
		})
	case ARCHIVE_FLAT:
		uidMap = make(map[string][]uint32)
		uidMap[archiveDir] = commands.UidsFromMessageInfos(msgs)
	}

	var wg sync.WaitGroup
	wg.Add(len(uidMap))
	success := true

	for dir, uids := range uidMap {
		store.Move(uids, dir, true, func(
			msg types.WorkerMessage,
		) {
			switch msg := msg.(type) {
			case *types.Done:
				wg.Done()
			case *types.Error:
				aerc.PushError(msg.Error.Error())
				success = false
				wg.Done()
				marker.Remark()
			}
		})
	}
	// we need to do that in the background, else we block the main thread
	go func() {
		defer log.PanicHandler()

		wg.Wait()
		if success {
			handleDone(aerc, acct, next, "Messages archived.", store)
		}
	}()
	return nil
}

func groupBy(msgs []*models.MessageInfo,
	grouper func(*models.MessageInfo) string,
) map[string][]uint32 {
	m := make(map[string][]uint32)
	for _, msg := range msgs {
		group := grouper(msg)
		m[group] = append(m[group], msg.Uid)
	}
	return m
}
