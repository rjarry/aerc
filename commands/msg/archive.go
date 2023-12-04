package msg

import (
	"fmt"
	"strings"
	"sync"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/log"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

const (
	ARCHIVE_FLAT  = "flat"
	ARCHIVE_YEAR  = "year"
	ARCHIVE_MONTH = "month"
)

var ARCHIVE_TYPES = []string{ARCHIVE_FLAT, ARCHIVE_YEAR, ARCHIVE_MONTH}

type Archive struct {
	Type string `opt:"type" action:"ParseArchiveType" metavar:"flat|year|month" complete:"CompleteType"`
}

func (a *Archive) ParseArchiveType(arg string) error {
	for _, t := range ARCHIVE_TYPES {
		if t == arg {
			a.Type = arg
			return nil
		}
	}
	return fmt.Errorf("invalid archive type")
}

func init() {
	commands.Register(Archive{})
}

func (Archive) Context() commands.CommandContext {
	return commands.MESSAGE
}

func (Archive) Aliases() []string {
	return []string{"archive"}
}

func (*Archive) CompleteType(arg string) []string {
	return commands.FilterList(ARCHIVE_TYPES, arg, nil)
}

func (a Archive) Execute(args []string) error {
	h := newHelper()
	msgs, err := h.messages()
	if err != nil {
		return err
	}
	err = archive(msgs, a.Type)
	return err
}

func archive(msgs []*models.MessageInfo, archiveType string) error {
	h := newHelper()
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
			dir := strings.Join([]string{
				archiveDir,
				fmt.Sprintf("%d", msg.Envelope.Date.Year()),
				fmt.Sprintf("%02d", msg.Envelope.Date.Month()),
			}, app.SelectedAccount().Worker().PathSeparator(),
			)
			return dir
		})
	case ARCHIVE_YEAR:
		uidMap = groupBy(msgs, func(msg *models.MessageInfo) string {
			dir := strings.Join([]string{
				archiveDir,
				fmt.Sprintf("%v", msg.Envelope.Date.Year()),
			}, app.SelectedAccount().Worker().PathSeparator(),
			)
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
				app.PushError(msg.Error.Error())
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
			var s string
			if len(uids) > 1 {
				s = "%d messages archived to %s"
			} else {
				s = "%d message archived to %s"
			}
			handleDone(acct, next, fmt.Sprintf(s, len(uids), archiveDir), store)
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
