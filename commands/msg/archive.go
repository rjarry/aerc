package msg

import (
	"errors"
	"fmt"
	"path"
	"time"

	"github.com/gdamore/tcell"

	"git.sr.ht/~sircmpwn/aerc/widgets"
	"git.sr.ht/~sircmpwn/aerc/worker/types"
)

const (
	ARCHIVE_FLAT  = "flat"
	ARCHIVE_YEAR  = "year"
	ARCHIVE_MONTH = "month"
)

func init() {
	register("archive", Archive)
}

func Archive(aerc *widgets.Aerc, args []string) error {
	if len(args) != 2 {
		return errors.New("Usage: archive <flat|year|month>")
	}
	acct := aerc.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	msg := acct.Messages().Selected()
	store := acct.Messages().Store()
	archiveDir := acct.AccountConfig().Archive
	acct.Messages().Next()

	switch args[1] {
	case ARCHIVE_MONTH:
		archiveDir = path.Join(archiveDir,
			fmt.Sprintf("%d", msg.Envelope.Date.Year()),
			fmt.Sprintf("%02d", msg.Envelope.Date.Month()))
	case ARCHIVE_YEAR:
		archiveDir = path.Join(archiveDir, fmt.Sprintf("%v",
			msg.Envelope.Date.Year()))
	case ARCHIVE_FLAT:
		// deliberately left blank
	}

	store.Move([]uint32{msg.Uid}, archiveDir, true, func(
		msg types.WorkerMessage) {

		switch msg := msg.(type) {
		case *types.Done:
			aerc.PushStatus("Messages archived.", 10*time.Second)
		case *types.Error:
			aerc.PushStatus(" "+msg.Error.Error(), 10*time.Second).
				Color(tcell.ColorDefault, tcell.ColorRed)
		}
	})
	return nil
}
