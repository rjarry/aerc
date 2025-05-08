package msg

import (
	"fmt"
	"iter"
	"slices"
	"strings"

	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

type Mark struct {
	All             bool   `opt:"-a" aliases:"mark,unmark" desc:"Mark all messages in current folder."`
	Toggle          bool   `opt:"-t" aliases:"mark,unmark" desc:"Toggle the marked state."`
	Visual          bool   `opt:"-v" aliases:"mark,unmark" desc:"Enter / leave visual mark mode."`
	VisualClear     bool   `opt:"-V" aliases:"mark,unmark" desc:"Same as -v but does not clear existing selection."`
	Thread          bool   `opt:"-T" aliases:"mark,unmark" desc:"Mark all messages from the selected thread."`
	SenderFilter    bool   `opt:"-s" aliases:"mark,unmark" desc:"Mark all messages having the substring in their From: header."`
	RecipientFilter bool   `opt:"-r" aliases:"mark,unmark" desc:"Mark all messages having the substring in their To: header."`
	FilterString    string `opt:"..." required:"false" desc:"Mark messages matching this string."`
}

func init() {
	commands.Register(Mark{})
}

func (Mark) Description() string {
	return "Mark, unmark or remark messages."
}

func (Mark) Context() commands.CommandContext {
	return commands.MESSAGE_LIST | commands.MESSAGE_VIEWER
}

func (Mark) Aliases() []string {
	return []string{"mark", "unmark", "remark"}
}

func (m Mark) Execute(args []string) error {
	h := newHelper()
	OnSelectedMessage := func(fn func(models.UID)) error {
		if fn == nil {
			return fmt.Errorf("no operation selected")
		}
		selected, err := h.msgProvider.SelectedMessage()
		if err != nil {
			return err
		}
		fn(selected.Uid)
		return nil
	}
	store, err := h.store()
	if err != nil {
		return err
	}
	marker := store.Marker()

	if m.Thread && m.All {
		return fmt.Errorf("-a and -T are mutually exclusive")
	}
	if m.Thread && (m.Visual || m.VisualClear) {
		return fmt.Errorf("-v and -T are mutually exclusive")
	}
	if m.Visual && m.All {
		return fmt.Errorf("-a and -v are mutually exclusive")
	}
	if m.SenderFilter && m.RecipientFilter {
		return fmt.Errorf("-s and -r are mutually exclusive")
	}
	if m.Visual && m.FilterString != "" {
		return fmt.Errorf("visual mode does not support filtering")
	}
	if (m.SenderFilter || m.RecipientFilter) && m.FilterString == "" {
		return fmt.Errorf("-s and -r require a filter string")
	}

	// if filtering and only a single message is provided, filter all
	// instead
	m.All = (m.FilterString != "" && !(m.Thread || m.All)) || m.All

	filter := slices.Values[[]models.UID, models.UID]
	switch {
	case m.SenderFilter:
		filter = senderFilter(store, m.FilterString)
	case m.RecipientFilter:
		filter = recipientFilter(store, m.FilterString)
	case m.FilterString != "":
		filter = subjectFilter(store, m.FilterString)
	}

	switch args[0] {
	case "mark":
		var modFunc func(models.UID)
		if m.Toggle {
			modFunc = marker.ToggleMark
		} else {
			modFunc = marker.Mark
		}
		switch {
		case m.All:
			uids := store.Uids()
			for uid := range filter(uids) {
				modFunc(uid)
			}
			return nil
		case m.Visual || m.VisualClear:
			marker.ToggleVisualMark(m.VisualClear)
			return nil
		default:
			if m.Thread {
				threadPtr, err := store.SelectedThread()
				if err != nil {
					return err
				}
				for uid := range filter(threadPtr.Root().Uids()) {
					modFunc(uid)
				}
			} else {
				return OnSelectedMessage(modFunc)
			}
			return nil
		}

	case "unmark":
		if m.Visual || m.VisualClear {
			return fmt.Errorf("visual mode not supported for this command")
		}

		switch {
		case m.All && m.Toggle:
			uids := store.Uids()
			for uid := range filter(uids) {
				marker.ToggleMark(uid)
			}
			return nil
		case m.All && !m.Toggle:
			marker.ClearVisualMark()
			return nil
		default:
			if m.Thread {
				threadPtr, err := store.SelectedThread()
				if err != nil {
					return err
				}
				for uid := range filter(threadPtr.Root().Uids()) {
					marker.Unmark(uid)
				}
			} else {
				return OnSelectedMessage(marker.Unmark)
			}
			return nil
		}
	case "remark":
		marker.Remark()
		return nil
	}
	return nil // never reached
}

func senderFilter(store *lib.MessageStore, senderMatches string) func([]models.UID) iter.Seq[models.UID] {
	return func(uids []models.UID) iter.Seq[models.UID] {
		store.FetchHeaders(uids, func(types.WorkerMessage) {})

		store.Lock()
		defer store.Unlock()

		var filteredUIDs []models.UID
		for _, uid := range uids {
			log.Debugf("checking for %s in messageStore", uid)
			msg := store.Messages[uid]
			if msg == nil {
				log.Warnf("message not found in messageStore")
				continue
			}
			log.Debugf("message: %#v", msg)
			from := msg.Envelope.From
			for _, sender := range from {
				if strings.Contains(sender.String(), senderMatches) {
					filteredUIDs = append(filteredUIDs, uid)
					break
				}
			}
		}

		return slices.Values(filteredUIDs)
	}
}

func recipientFilter(store *lib.MessageStore, recipientMatches string) func([]models.UID) iter.Seq[models.UID] {
	return func(uids []models.UID) iter.Seq[models.UID] {
		store.FetchHeaders(uids, func(types.WorkerMessage) {})

		store.Lock()
		defer store.Unlock()

		var filteredUIDs []models.UID
		for _, uid := range uids {
			log.Debugf("checking for %s in messageStore", uid)
			msg := store.Messages[uid]
			if msg == nil {
				log.Warnf("message not found in messageStore")
				continue
			}
			log.Debugf("message: %#v", msg)
			recipients := msg.Envelope.To
			for _, recipient := range recipients {
				if strings.Contains(recipient.String(), recipientMatches) {
					filteredUIDs = append(filteredUIDs, uid)
					break
				}
			}
		}

		return slices.Values(filteredUIDs)
	}
}

func subjectFilter(store *lib.MessageStore, subjectMatches string) func([]models.UID) iter.Seq[models.UID] {
	return func(uids []models.UID) iter.Seq[models.UID] {
		store.FetchHeaders(uids, func(types.WorkerMessage) {})

		store.Lock()
		defer store.Unlock()

		var filteredUIDs []models.UID
		for _, uid := range uids {
			log.Debugf("checking for %s in messageStore", uid)
			msg := store.Messages[uid]
			if msg == nil {
				log.Warnf("message not found in messageStore")
				continue
			}
			log.Debugf("message: %#v", msg)
			subject := msg.Envelope.Subject
			if strings.Contains(subject, subjectMatches) {
				filteredUIDs = append(filteredUIDs, uid)
			}
		}

		return slices.Values(filteredUIDs)
	}
}
